package mongodb

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/chzyer/readline"
)

// REPL is an interactive MongoDB session.
type REPL struct {
	client *Client
}

// NewREPL creates a MongoDB REPL for the given client.
func NewREPL(client *Client) *REPL {
	return &REPL{client: client}
}

// Run starts the interactive readline loop.
func (r *REPL) Run() error {
	histFile := os.ExpandEnv("$HOME/.db-cli_mongo_history")
	rl, err := readline.NewEx(&readline.Config{
		Prompt:            r.prompt(),
		HistoryFile:       histFile,
		InterruptPrompt:   "^C",
		EOFPrompt:         "exit",
		HistorySearchFold: true,
	})
	if err != nil {
		return fmt.Errorf("readline init: %w", err)
	}
	defer rl.Close()

	ctx := context.Background()
	fmt.Printf("Connected to MongoDB @ %s:%d (db: %s)\n", r.client.Host, r.client.Port, r.client.DBName)
	fmt.Println("Enter JSON commands (e.g. {\"find\":\"users\",\"filter\":{}}), \\h for help, \\q to quit.")

	var buf strings.Builder

	for {
		multiline := buf.Len() > 0
		if multiline {
			rl.SetPrompt("    ... ")
		} else {
			rl.SetPrompt(r.prompt())
		}

		line, err := rl.Readline()
		if err != nil {
			break
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if !multiline && strings.HasPrefix(line, "\\") {
			if r.handleMeta(ctx, rl, line) {
				break
			}
			continue
		}
		if !multiline && (strings.EqualFold(line, "exit") || strings.EqualFold(line, "quit")) {
			break
		}

		buf.WriteString(line)
		// Execute when braces are balanced.
		if isComplete(buf.String()) {
			r.exec(ctx, buf.String())
			buf.Reset()
		}
	}
	fmt.Println("Bye.")
	return nil
}

func (r *REPL) handleMeta(ctx context.Context, rl *readline.Instance, cmd string) (quit bool) {
	parts := strings.Fields(cmd)
	switch parts[0] {
	case "\\q", "\\quit":
		return true
	case "\\h", "\\help":
		printHelp()
	case "\\d":
		names, err := r.client.ListDatabases(ctx)
		if err != nil {
			fmt.Printf("ERROR: %v\n", err)
		} else {
			for _, n := range names {
				fmt.Println(n)
			}
		}
	case "\\dt":
		names, err := r.client.ListCollections(ctx)
		if err != nil {
			fmt.Printf("ERROR: %v\n", err)
		} else {
			for _, n := range names {
				fmt.Println(n)
			}
		}
	case "\\c":
		if len(parts) < 2 {
			fmt.Printf("Current database: %s\n", r.client.DBName)
			return false
		}
		r.client.UseDB(parts[1])
		fmt.Printf("Switched to database %q\n", parts[1])
		rl.SetPrompt(r.prompt())
	default:
		fmt.Printf("Unknown command %q. Type \\h for help.\n", parts[0])
	}
	return false
}

func (r *REPL) exec(ctx context.Context, cmdStr string) {
	result, err := r.client.RunCommand(ctx, cmdStr)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return
	}
	fmt.Println(result)
}

func (r *REPL) prompt() string {
	return fmt.Sprintf("[mongo@%s:%d/%s]> ", r.client.Host, r.client.Port, r.client.DBName)
}

// isComplete returns true when the JSON input has balanced braces.
func isComplete(s string) bool {
	depth := 0
	inStr := false
	for i, ch := range s {
		if inStr {
			if ch == '"' && (i == 0 || s[i-1] != '\\') {
				inStr = false
			}
			continue
		}
		switch ch {
		case '"':
			inStr = true
		case '{':
			depth++
		case '}':
			depth--
		}
	}
	return depth == 0 && strings.ContainsAny(s, "{}")
}

func printHelp() {
	fmt.Print(`
Meta-commands:
  \q, \quit          Exit
  \h, \help          Show this help
  \d                 List databases
  \dt                List collections in current database
  \c [DBNAME]        Switch to database (show current if omitted)

Enter MongoDB commands as JSON (runCommand format):
  {"find": "users", "filter": {"age": {"$gt": 18}}, "limit": 10}
  {"insert": "logs", "documents": [{"msg": "hello"}]}
  {"drop": "oldcollection"}

Multi-line input is supported — keep typing until braces are balanced.

`)
}
