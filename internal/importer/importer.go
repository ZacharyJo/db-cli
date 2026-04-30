package importer

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/schollz/progressbar/v3"
	"github.com/ZacharyJo/db-cli/internal/config"
)

// OnConflict controls how INSERT conflicts are handled.
type OnConflict string

const (
	OnConflictDefault OnConflict = ""       // keep INSERT as-is
	OnConflictIgnore  OnConflict = "ignore" // rewrite to INSERT IGNORE / ON CONFLICT DO NOTHING
)

// Options controls importer behavior.
type Options struct {
	BatchSize    int        // commit every N statements (0 = no explicit commit batching)
	StopOnError  bool       // abort on first error
	Verbose      bool       // print each statement before executing
	OnConflict   OnConflict // conflict resolution strategy for INSERT
	IgnoreErrors bool       // skip all errors and continue (errors still reported at end)
	PgWire       bool       // true when target is PostgreSQL-wire (affects INSERT rewrite syntax)
	CreateDB     bool       // auto-create databases encountered in USE statements
	Cfg          *config.Config // used to reconnect on USE dbname
	OpenDB       func(cfg *config.Config) (*sql.DB, error) // injected by caller
	EnsureDB     func(cfg *config.Config) error            // injected by caller (optional)
}

// Import streams-parses sqlFile and executes each statement on db.
// It returns the number of successfully executed statements and any accumulated errors.
// All statements are executed on a single dedicated connection so that session-level
// state (e.g. SET search_path) is preserved across statements.
// When a USE dbname statement is encountered, the connection is switched to that database.
func Import(db *sql.DB, sqlFile string, opts Options) (int, []error) {
	f, err := os.Open(sqlFile)
	if err != nil {
		return 0, []error{fmt.Errorf("open %s: %w", sqlFile, err)}
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return 0, []error{fmt.Errorf("stat %s: %w", sqlFile, err)}
	}

	bar := progressbar.NewOptions64(
		fi.Size(),
		progressbar.OptionSetDescription("importing"),
		progressbar.OptionShowBytes(true),
		progressbar.OptionShowCount(),
		progressbar.OptionSetWidth(40),
		progressbar.OptionClearOnFinish(),
	)

	pr := &progressReader{r: f, bar: bar}
	count, errs := execStatements(db, pr, opts)
	fmt.Println() // newline after progress bar
	return count, errs
}

// progressReader wraps an io.Reader and updates a progress bar on each read.
type progressReader struct {
	r   io.Reader
	bar *progressbar.ProgressBar
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.r.Read(p)
	if n > 0 {
		pr.bar.Add(n) //nolint:errcheck
	}
	return n, err
}

// execStatements parses SQL statements from r and executes them on conn.
// When a USE dbname statement is found, it reconnects to the new database.
func execStatements(db *sql.DB, r io.Reader, opts Options) (int, []error) {
	var (
		errs      []error
		count     int
		delimiter = ";"
		currentDB = db // current pool; may be replaced on USE
		fatalErr  bool // set by flushStmt when a non-ignorable error occurs
	)

	ctx := context.Background()

	// acquireConn creates a dedicated connection from the given pool.
	acquireConn := func(pool *sql.DB) (*sql.Conn, error) {
		return pool.Conn(ctx)
	}

	conn, err := acquireConn(currentDB)
	if err != nil {
		return 0, []error{fmt.Errorf("acquire connection: %w", err)}
	}
	defer func() { conn.Close() }()

	// switchDB reconnects to a different database. Called when USE dbname is seen.
	switchDB := func(dbname string) error {
		if opts.OpenDB == nil || opts.Cfg == nil {
			// No reconnect capability — skip with a verbose hint.
			if opts.Verbose {
				fmt.Printf("SKIP> USE %s (no reconnect configured)\n", dbname)
			}
			return nil
		}
		newCfg := *opts.Cfg
		newCfg.Database = dbname
		// Optionally create the database if it doesn't exist.
		if opts.CreateDB && opts.EnsureDB != nil {
			if err := opts.EnsureDB(&newCfg); err != nil {
				return fmt.Errorf("ensure database %q: %w", dbname, err)
			}
		}
		newDB, err := opts.OpenDB(&newCfg)
		if err != nil {
			return fmt.Errorf("connect to database %q: %w", dbname, err)
		}
		// Close old connection and pool (if it differs from the original db).
		conn.Close()
		if currentDB != db {
			currentDB.Close()
		}
		currentDB = newDB
		newConn, err := acquireConn(newDB)
		if err != nil {
			// newDB is open but unusable — close it and restore invariant.
			newDB.Close()
			currentDB = db
			return fmt.Errorf("acquire connection for %q: %w", dbname, err)
		}
		conn = newConn
		fmt.Printf("Switched to database %q\n", dbname)
		return nil
	}

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1<<20), 1<<20) // 1 MiB line buffer

	var stmt strings.Builder
	inSingleQuote := false
	inDoubleQuote := false
	inLineComment := false
	inBlockComment := false

	flushStmt := func() {
		s := strings.TrimSpace(stmt.String())
		stmt.Reset()
		if s == "" {
			return
		}
		// Skip client meta-commands (e.g. \c, \connect, \i) — these are
		// psql/ksql directives that the server never understands.
		if strings.HasPrefix(s, "\\") {
			if opts.Verbose {
				fmt.Printf("SKIP> %s\n", s)
			}
			return
		}
		upper := strings.ToUpper(strings.TrimSpace(s))
		// Skip CREATE DATABASE — handled by --create-db flag or ignored.
		if strings.HasPrefix(upper, "CREATE DATABASE") {
			if opts.Verbose {
				fmt.Printf("SKIP> %s\n", s)
			}
			return
		}
		// Handle USE dbname — switch connection to the new database.
		if strings.HasPrefix(upper, "USE ") {
			dbname := strings.Trim(strings.TrimSpace(s[4:]), "`\"' \t\n;")
			if dbname != "" {
				if err := switchDB(dbname); err != nil {
					errs = append(errs, fmt.Errorf("switch db error: %w", err))
					if !opts.IgnoreErrors {
						fatalErr = true
					}
				}
			}
			return
		}
		// Rewrite CREATE TABLE / CREATE INDEX to IF NOT EXISTS when ignoring conflicts.
		if opts.OnConflict == OnConflictIgnore {
			s = rewriteCreate(s, upper)
		}
		// Rewrite INSERT for conflict handling.
		if opts.OnConflict == OnConflictIgnore {
			s = rewriteInsert(s, opts.PgWire)
		}
		if opts.Verbose {
			fmt.Printf("SQL> %s\n", s)
		}
		if _, err := conn.ExecContext(ctx, s); err != nil {
			if opts.IgnoreErrors {
				errs = append(errs, fmt.Errorf("exec error: %w\nStatement: %s", err, s))
				return
			}
			errs = append(errs, fmt.Errorf("exec error: %w\nStatement: %s", err, s))
			return
		}
		count++
	}

	for scanner.Scan() {
		line := scanner.Text()

		// Handle DELIMITER directive (MySQL stored procedures).
		if upper := strings.ToUpper(strings.TrimSpace(line)); strings.HasPrefix(upper, "DELIMITER ") {
			newDelim := strings.TrimSpace(line[len("DELIMITER "):])
			if newDelim != "" {
				delimiter = newDelim
			}
			continue
		}

		// Skip client meta-command lines (\c, \connect, \i, etc.) that appear
		// as standalone lines without a delimiter. These are psql/ksql directives
		// and must not be accumulated into the SQL buffer.
		if trimmed := strings.TrimSpace(line); strings.HasPrefix(trimmed, "\\") && stmt.Len() == 0 {
			if opts.Verbose {
				fmt.Printf("SKIP> %s\n", trimmed)
			}
			continue
		}

		// Reset per-line comment flag.
		inLineComment = false

		for i := 0; i < len(line); i++ {
			ch := line[i]

			if inBlockComment {
				if ch == '*' && i+1 < len(line) && line[i+1] == '/' {
					inBlockComment = false
					i++ // skip '/'
				}
				continue
			}

			if inLineComment {
				break // rest of line is comment
			}

			if inSingleQuote {
				stmt.WriteByte(ch)
				if ch == '\'' {
					// Handle escaped quote: ''
					if i+1 < len(line) && line[i+1] == '\'' {
						stmt.WriteByte('\'')
						i++
					} else {
						inSingleQuote = false
					}
				}
				continue
			}

			if inDoubleQuote {
				stmt.WriteByte(ch)
				if ch == '"' {
					inDoubleQuote = false
				}
				continue
			}

			// Detect comment start.
			if ch == '-' && i+1 < len(line) && line[i+1] == '-' {
				inLineComment = true
				break
			}
			if ch == '/' && i+1 < len(line) && line[i+1] == '*' {
				inBlockComment = true
				i++
				continue
			}

			if ch == '\'' {
				inSingleQuote = true
				stmt.WriteByte(ch)
				continue
			}
			if ch == '"' {
				inDoubleQuote = true
				stmt.WriteByte(ch)
				continue
			}

			// Check for delimiter match at this position.
			if strings.HasPrefix(line[i:], delimiter) {
				flushStmt()
				i += len(delimiter) - 1 // -1 because loop will do i++
				if (opts.StopOnError && !opts.IgnoreErrors && len(errs) > 0) || fatalErr {
					return count, errs
				}
				continue
			}

			stmt.WriteByte(ch)
		}

		// Preserve newlines within multi-line statements.
		if stmt.Len() > 0 {
			stmt.WriteByte('\n')
		}

		// A USE statement sets fatalErr without a delimiter — check after each line.
		if fatalErr {
			return count, errs
		}
	}

	// Flush any remaining statement without trailing delimiter.
	if s := strings.TrimSpace(stmt.String()); s != "" {
		flushStmt()
	}

	// Close extra pools opened for USE switches.
	if currentDB != db {
		currentDB.Close()
	}

	if err := scanner.Err(); err != nil {
		errs = append(errs, fmt.Errorf("scan error: %w", err))
	}
	return count, errs
}

// rewriteCreate rewrites CREATE TABLE and CREATE INDEX to their IF NOT EXISTS variant.
func rewriteCreate(s, upper string) string {
	for _, prefix := range []string{"CREATE TABLE ", "CREATE UNIQUE INDEX ", "CREATE INDEX "} {
		if strings.HasPrefix(upper, prefix) {
			rest := upper[len(prefix):]
			if strings.HasPrefix(rest, "IF NOT EXISTS") {
				return s
			}
			return s[:len(prefix)] + "IF NOT EXISTS " + s[len(prefix):]
		}
	}
	return s
}
// MySQL wire: INSERT INTO → INSERT IGNORE INTO
// PG wire:    appends ON CONFLICT DO NOTHING
func rewriteInsert(s string, pgWire bool) string {
	upper := strings.ToUpper(strings.TrimSpace(s))
	if !strings.HasPrefix(upper, "INSERT") {
		return s
	}
	if pgWire {
		// Already has ON CONFLICT clause — leave it alone.
		if strings.Contains(upper, "ON CONFLICT") {
			return s
		}
		trimmed := strings.TrimRight(strings.TrimSpace(s), ";")
		return trimmed + " ON CONFLICT DO NOTHING"
	}
	// MySQL: INSERT INTO → INSERT IGNORE INTO (case-preserving)
	idx := strings.Index(upper, "INSERT")
	after := strings.TrimSpace(s[idx+len("INSERT"):])
	afterUpper := strings.ToUpper(after)
	// Already has IGNORE — leave it alone.
	if strings.HasPrefix(afterUpper, "IGNORE") {
		return s
	}
	if strings.HasPrefix(afterUpper, "INTO") {
		return s[:idx] + "INSERT IGNORE INTO" + after[len("INTO"):]
	}
	return s[:idx] + "INSERT IGNORE" + s[idx+len("INSERT"):]
}
