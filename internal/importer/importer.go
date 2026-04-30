package importer

import (
	"bufio"
	"database/sql"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/schollz/progressbar/v3"
)

// Options controls importer behavior.
type Options struct {
	BatchSize   int  // commit every N statements (0 = no explicit commit batching)
	StopOnError bool // abort on first error
	Verbose     bool // print each statement before executing
}

// Import streams-parses sqlFile and executes each statement on db.
// It returns the number of successfully executed statements and any accumulated errors.
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

// execStatements parses SQL statements from r and executes them.
func execStatements(db *sql.DB, r io.Reader, opts Options) (int, []error) {
	var (
		errs      []error
		count     int
		delimiter = ";"
	)

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
		if opts.Verbose {
			fmt.Printf("SQL> %s\n", s)
		}
		if _, err := db.Exec(s); err != nil {
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
				if opts.StopOnError && len(errs) > 0 {
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
	}

	// Flush any remaining statement without trailing delimiter.
	if s := strings.TrimSpace(stmt.String()); s != "" {
		flushStmt()
	}

	if err := scanner.Err(); err != nil {
		errs = append(errs, fmt.Errorf("scan error: %w", err))
	}
	return count, errs
}
