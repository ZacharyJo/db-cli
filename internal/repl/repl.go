package repl

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/chzyer/readline"
	"github.com/ZacharyJo/mysql-cli-go/internal/db"
	"github.com/ZacharyJo/mysql-cli-go/internal/output"
)

// REPL holds state for an interactive SQL session.
type REPL struct {
	conn    *db.Connector
	printer *output.Printer
	timing  bool
	dbType  string
	host    string
	dbName  string
}

// New creates a new REPL connected to the given Connector.
func New(conn *db.Connector, dbType, host, dbName, format string) *REPL {
	return &REPL{
		conn:    conn,
		printer: output.New(format),
		dbType:  dbType,
		host:    host,
		dbName:  dbName,
	}
}

// Run starts the interactive readline loop. Returns when the user exits.
func (r *REPL) Run() error {
	histFile := os.ExpandEnv("$HOME/.mysqlcli_history")
	rl, err := readline.NewEx(&readline.Config{
		Prompt:            r.prompt(false),
		HistoryFile:       histFile,
		InterruptPrompt:   "^C",
		EOFPrompt:         "exit",
		HistorySearchFold: true,
	})
	if err != nil {
		return fmt.Errorf("readline init: %w", err)
	}
	defer rl.Close()

	var buf strings.Builder

	fmt.Printf("Connected to %s @ %s (database: %s)\n", r.dbType, r.host, r.dbName)
	fmt.Println("Type \\h for help, \\q to quit.")

	for {
		multiline := buf.Len() > 0
		rl.SetPrompt(r.prompt(multiline))
		line, err := rl.Readline()
		if err != nil { // EOF or ^D
			break
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Handle meta-commands.
		if strings.HasPrefix(line, "\\") {
			if r.handleMeta(line) {
				break // \q
			}
			buf.Reset()
			continue
		}
		if strings.EqualFold(line, "exit") || strings.EqualFold(line, "quit") {
			break
		}

		buf.WriteString(line)
		buf.WriteByte('\n')

		// Execute once we see a ';' at the end of the accumulated buffer.
		if strings.Contains(line, ";") {
			sql := strings.TrimRight(strings.TrimSpace(buf.String()), ";")
			buf.Reset()
			if sql == "" {
				continue
			}
			r.execute(sql)
		}
	}
	fmt.Println("Bye.")
	return nil
}

// handleMeta processes backslash meta-commands. Returns true if the REPL should exit.
func (r *REPL) handleMeta(cmd string) (quit bool) {
	parts := strings.Fields(cmd)
	switch parts[0] {
	case "\\q", "\\quit":
		return true
	case "\\h", "\\help":
		printHelp()
	case "\\timing":
		r.timing = !r.timing
		if r.timing {
			fmt.Println("Timing is on.")
		} else {
			fmt.Println("Timing is off.")
		}
	case "\\output":
		if len(parts) < 2 {
			fmt.Println("Usage: \\output [table|json|csv]")
			return false
		}
		switch parts[1] {
		case output.FormatTable, output.FormatJSON, output.FormatCSV:
			r.printer.Format = parts[1]
			fmt.Printf("Output format: %s\n", parts[1])
		default:
			fmt.Printf("Unknown format %q. Use table, json, or csv.\n", parts[1])
		}
	case "\\d":
		r.showDatabases()
	case "\\e":
		r.openEditor()
	default:
		fmt.Printf("Unknown command %q. Type \\h for help.\n", parts[0])
	}
	return false
}

func (r *REPL) execute(sql string) {
	start := time.Now()

	if db.IsReadQuery(sql) {
		rows, err := r.conn.QueryDB().Query(sql)
		elapsed := time.Since(start)
		if err != nil {
			r.printer.PrintError(err)
			return
		}
		defer rows.Close()
		if err := r.printer.PrintRows(rows, elapsed); err != nil {
			r.printer.PrintError(err)
		}
	} else {
		res, err := r.conn.WriteDB().Exec(sql)
		elapsed := time.Since(start)
		if err != nil {
			r.printer.PrintError(err)
			return
		}
		r.printer.PrintResult(res, elapsed)
	}
}

func (r *REPL) showDatabases() {
	var query string
	if r.dbType == "gaussdb" || r.dbType == "kingbase" {
		query = "SELECT datname FROM pg_database ORDER BY datname"
	} else {
		query = "SHOW DATABASES"
	}
	r.execute(query)
}

func (r *REPL) openEditor() {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	tmpf, err := os.CreateTemp("", "mysqlcli-*.sql")
	if err != nil {
		fmt.Println("ERROR: cannot create temp file:", err)
		return
	}
	tmpf.Close()
	defer os.Remove(tmpf.Name())

	cmd := exec.Command(editor, tmpf.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Println("ERROR: editor exited:", err)
		return
	}
	content, err := os.ReadFile(tmpf.Name())
	if err != nil || len(strings.TrimSpace(string(content))) == 0 {
		return
	}
	sql := strings.TrimRight(strings.TrimSpace(string(content)), ";")
	fmt.Printf("-- executing:\n%s\n", sql)
	r.execute(sql)
}

func (r *REPL) prompt(multiline bool) string {
	if multiline {
		return "    -> "
	}
	return fmt.Sprintf("[%s@%s/%s]> ", r.dbType, r.host, r.dbName)
}

func printHelp() {
	fmt.Print(`
Meta-commands:
  \q, \quit          Exit
  \h, \help          Show this help
  \d                 Show databases
  \timing            Toggle query timing
  \output FORMAT     Set output format: table | json | csv
  \e                 Open $EDITOR to compose SQL
  exit, quit         Alias for \q

Terminate SQL statements with ; to execute.
Multi-line input is supported — keep typing until ;

`)
}
