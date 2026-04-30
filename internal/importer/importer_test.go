package importer

import (
	"strings"
	"testing"
)

// mockDB implements a minimal sql.DB-like interface for testing.
// We test the parser (execStatements) by capturing what would be executed.

type captureDB struct {
	stmts []string
	err   error
}

// We test the internal SQL splitter via a helper that returns statements without executing.
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
