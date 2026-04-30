package importer

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// SQL 分割器测试（已有，保留）
// ---------------------------------------------------------------------------

func splitStatements(input string) []string {
	var stmts []string
	var stmt strings.Builder
	delimiter := ";"
	inSingle := false
	inDouble := false
	inLine := false
	inBlock := false

	lines := strings.Split(input, "\n")
	for _, line := range lines {
		inLine = false
		for i := 0; i < len(line); i++ {
			ch := line[i]
			if inBlock {
				if ch == '*' && i+1 < len(line) && line[i+1] == '/' {
					inBlock = false
					i++
				}
				continue
			}
			if inLine {
				break
			}
			if inSingle {
				stmt.WriteByte(ch)
				if ch == '\'' {
					if i+1 < len(line) && line[i+1] == '\'' {
						stmt.WriteByte('\'')
						i++
					} else {
						inSingle = false
					}
				}
				continue
			}
			if inDouble {
				stmt.WriteByte(ch)
				if ch == '"' {
					inDouble = false
				}
				continue
			}
			if ch == '-' && i+1 < len(line) && line[i+1] == '-' {
				inLine = true
				break
			}
			if ch == '/' && i+1 < len(line) && line[i+1] == '*' {
				inBlock = true
				i++
				continue
			}
			if ch == '\'' {
				inSingle = true
				stmt.WriteByte(ch)
				continue
			}
			if ch == '"' {
				inDouble = true
				stmt.WriteByte(ch)
				continue
			}
			if strings.HasPrefix(line[i:], delimiter) {
				s := strings.TrimSpace(stmt.String())
				if s != "" {
					stmts = append(stmts, s)
				}
				stmt.Reset()
				i += len(delimiter) - 1
				continue
			}
			stmt.WriteByte(ch)
		}
		if stmt.Len() > 0 {
			stmt.WriteByte('\n')
		}
	}
	if s := strings.TrimSpace(stmt.String()); s != "" {
		stmts = append(stmts, s)
	}
	return stmts
}

func TestSplitSimple(t *testing.T) {
	input := "SELECT 1; SELECT 2;"
	stmts := splitStatements(input)
	if len(stmts) != 2 {
		t.Fatalf("expected 2 statements, got %d: %v", len(stmts), stmts)
	}
	if stmts[0] != "SELECT 1" {
		t.Errorf("stmt[0] = %q", stmts[0])
	}
	if stmts[1] != "SELECT 2" {
		t.Errorf("stmt[1] = %q", stmts[1])
	}
}

func TestSplitQuotedSemicolon(t *testing.T) {
	input := `INSERT INTO t VALUES('hello; world');`
	stmts := splitStatements(input)
	if len(stmts) != 1 {
		t.Fatalf("expected 1 statement, got %d: %v", len(stmts), stmts)
	}
	if !strings.Contains(stmts[0], "hello; world") {
		t.Errorf("quoted semicolon was wrongly split: %s", stmts[0])
	}
}

func TestSplitLineComment(t *testing.T) {
	input := "SELECT 1 -- this is a comment\n; SELECT 2;"
	stmts := splitStatements(input)
	if len(stmts) != 2 {
		t.Fatalf("expected 2 statements, got %d: %v", len(stmts), stmts)
	}
}

func TestSplitBlockComment(t *testing.T) {
	input := "SELECT /* comment */ 1; SELECT 2;"
	stmts := splitStatements(input)
	if len(stmts) != 2 {
		t.Fatalf("expected 2 statements, got %d: %v", len(stmts), stmts)
	}
}

func TestSplitMultiline(t *testing.T) {
	input := "SELECT\n  id,\n  name\nFROM users\nWHERE id = 1;"
	stmts := splitStatements(input)
	if len(stmts) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(stmts))
	}
	if !strings.Contains(stmts[0], "FROM users") {
		t.Errorf("multi-line statement not preserved: %s", stmts[0])
	}
}

func TestSplitEscapedQuote(t *testing.T) {
	input := `INSERT INTO t VALUES('it''s here; yes');`
	stmts := splitStatements(input)
	if len(stmts) != 1 {
		t.Fatalf("expected 1 statement, got %d: %v", len(stmts), stmts)
	}
}

// ---------------------------------------------------------------------------
// rewriteInsert 测试
// ---------------------------------------------------------------------------

func TestRewriteInsert_MySQL(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{
			`INSERT INTO "t" ("id") VALUES (1)`,
			`INSERT IGNORE INTO "t" ("id") VALUES (1)`,
		},
		{
			`insert into t (id) values (2)`,
			`INSERT IGNORE INTO t (id) values (2)`,
		},
		{
			// 已有 IGNORE，不应重复
			`INSERT IGNORE INTO t (id) VALUES (3)`,
			`INSERT IGNORE INTO t (id) VALUES (3)`,
		},
		{
			// 非 INSERT，不改写
			`UPDATE t SET a=1`,
			`UPDATE t SET a=1`,
		},
		{
			`CREATE TABLE t (id INT)`,
			`CREATE TABLE t (id INT)`,
		},
	}
	for _, c := range cases {
		got := rewriteInsert(c.in, false)
		if got != c.want {
			t.Errorf("rewriteInsert(%q, mysql)\n  want: %q\n  got:  %q", c.in, c.want, got)
		}
	}
}

func TestRewriteInsert_PG(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{
			`INSERT INTO "t" ("id") VALUES (1)`,
			`INSERT INTO "t" ("id") VALUES (1) ON CONFLICT DO NOTHING`,
		},
		{
			// 末尾有分号，分号应被去掉再追加
			`INSERT INTO "t" ("id") VALUES (1);`,
			`INSERT INTO "t" ("id") VALUES (1) ON CONFLICT DO NOTHING`,
		},
		{
			// 非 INSERT，不改写
			`UPDATE t SET a=1`,
			`UPDATE t SET a=1`,
		},
	}
	for _, c := range cases {
		got := rewriteInsert(c.in, true)
		if got != c.want {
			t.Errorf("rewriteInsert(%q, pg)\n  want: %q\n  got:  %q", c.in, c.want, got)
		}
	}
}

// ---------------------------------------------------------------------------
// rewriteCreate 测试
// ---------------------------------------------------------------------------

func TestRewriteCreate_Table(t *testing.T) {
	in := `CREATE TABLE "t_foo" (id INT)`
	upper := strings.ToUpper(strings.TrimSpace(in))
	got := rewriteCreate(in, upper)
	want := `CREATE TABLE IF NOT EXISTS "t_foo" (id INT)`
	if got != want {
		t.Errorf("rewriteCreate table\n  want: %q\n  got:  %q", want, got)
	}
}

func TestRewriteCreate_TableAlreadyIfNotExists(t *testing.T) {
	in := `CREATE TABLE IF NOT EXISTS t (id INT)`
	upper := strings.ToUpper(strings.TrimSpace(in))
	got := rewriteCreate(in, upper)
	if got != in {
		t.Errorf("should be unchanged, got: %q", got)
	}
}

func TestRewriteCreate_Index(t *testing.T) {
	in := `CREATE INDEX idx_foo ON t (col)`
	upper := strings.ToUpper(strings.TrimSpace(in))
	got := rewriteCreate(in, upper)
	want := `CREATE INDEX IF NOT EXISTS idx_foo ON t (col)`
	if got != want {
		t.Errorf("rewriteCreate index\n  want: %q\n  got:  %q", want, got)
	}
}

func TestRewriteCreate_UniqueIndex(t *testing.T) {
	in := `CREATE UNIQUE INDEX uq_foo ON t (col)`
	upper := strings.ToUpper(strings.TrimSpace(in))
	got := rewriteCreate(in, upper)
	want := `CREATE UNIQUE INDEX IF NOT EXISTS uq_foo ON t (col)`
	if got != want {
		t.Errorf("rewriteCreate unique index\n  want: %q\n  got:  %q", want, got)
	}
}

func TestRewriteCreate_NonCreate(t *testing.T) {
	in := `ALTER TABLE t ADD COLUMN x INT`
	upper := strings.ToUpper(strings.TrimSpace(in))
	got := rewriteCreate(in, upper)
	if got != in {
		t.Errorf("non-create should be unchanged, got: %q", got)
	}
}
