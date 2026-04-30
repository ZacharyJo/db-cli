package redisdb

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/chzyer/readline"
)

// REPL is an interactive Redis command session.
type REPL struct {
	client *Client
}

// NewREPL creates a Redis REPL for the given client.
func NewREPL(client *Client) *REPL {
	return &REPL{client: client}
}

// Run starts the interactive readline loop.
func (r *REPL) Run() error {
	histFile := os.ExpandEnv("$HOME/.db-cli_redis_history")
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
	fmt.Printf("Connected to Redis [%s] @ %s (db %d)\n",
		r.client.CurrentMode(), r.client.AddrString(), r.client.CurrentDB())
	fmt.Println("Type commands directly (e.g. GET key), \\h for help, \\q to quit.")

	for {
		rl.SetPrompt(r.prompt())
		line, err := rl.Readline()
		if err != nil {
			break
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "\\") {
			if r.handleMeta(ctx, rl, line) {
				break
			}
			continue
		}
		if strings.EqualFold(line, "exit") || strings.EqualFold(line, "quit") {
			break
		}

		r.exec(ctx, line)
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
		info, err := r.client.Keyspace(ctx)
		if err != nil {
			fmt.Printf("ERROR: %v\n", err)
		} else {
			fmt.Print(info)
		}
	case "\\c":
		if len(parts) < 2 {
			fmt.Printf("Current DB: %d (mode: %s)\n", r.client.CurrentDB(), r.client.CurrentMode())
			return false
		}
		var db int
		if _, err := fmt.Sscanf(parts[1], "%d", &db); err != nil {
			fmt.Printf("ERROR: DB number must be an integer, got %q\n", parts[1])
			return false
		}
		if err := r.client.SelectDB(ctx, db); err != nil {
			fmt.Printf("ERROR: %v\n", err)
		} else {
			fmt.Printf("Switched to DB %d\n", db)
			rl.SetPrompt(r.prompt())
		}
	default:
		fmt.Printf("Unknown command %q. Type \\h for help.\n", parts[0])
	}
	return false
}

func (r *REPL) exec(ctx context.Context, line string) {
	parts := SplitArgs(line)
	if len(parts) == 0 {
		return
	}
	args := make([]interface{}, len(parts))
	for i, p := range parts {
		args[i] = p
	}
	result, err := r.client.Do(ctx, args...)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return
	}
	fmt.Println(result)
}

func (r *REPL) prompt() string {
	return fmt.Sprintf("[redis@%s/%d]> ", r.client.AddrString(), r.client.CurrentDB())
}

// SplitArgs splits a command line into tokens, respecting double-quoted strings.
func SplitArgs(line string) []string {
	var args []string
	var cur strings.Builder
	inQuote := false
	for _, ch := range line {
		switch {
		case ch == '"':
			inQuote = !inQuote
		case ch == ' ' && !inQuote:
			if cur.Len() > 0 {
				args = append(args, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteRune(ch)
		}
	}
	if cur.Len() > 0 {
		args = append(args, cur.String())
	}
	return args
}

func printHelp() {
	fmt.Print(`
Meta-commands:
  \q, \quit          Exit
  \h, \help          Show this help
  \d                 Show keyspace info (INFO keyspace)
  \c [N]             Switch to Redis DB number N (show current if omitted)

Type Redis commands directly (e.g. SET key val, GET key, HGETALL hash).

`)
}
