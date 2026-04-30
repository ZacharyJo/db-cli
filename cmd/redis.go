package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/ZacharyJo/db-cli/internal/redisdb"
)

var (
	redisHost     string
	redisPort     int
	redisPassword string
	redisDB       int
)

var redisCmd = &cobra.Command{
	Use:   "redis",
	Short: "Connect to Redis and open an interactive session",
	Example: `  db-cli redis -H 127.0.0.1 -P 6379
  db-cli redis -H 10.0.0.1 -P 6379 --password secret --db 1
  db-cli redis exec -H 127.0.0.1 "GET mykey"`,
	RunE: runRedisConnect,
}

var redisExecCmd = &cobra.Command{
	Use:   "exec COMMAND [ARGS...]",
	Short: "Execute a single Redis command and print the result",
	Example: `  db-cli redis exec -H 127.0.0.1 "GET mykey"
  db-cli redis exec -H 127.0.0.1 SET foo bar`,
	Args: cobra.MinimumNArgs(1),
	RunE: runRedisExec,
}

func init() {
	rootCmd.AddCommand(redisCmd)
	redisCmd.AddCommand(redisExecCmd)

	for _, cmd := range []*cobra.Command{redisCmd, redisExecCmd} {
		cmd.Flags().StringVarP(&redisHost, "host", "H", "127.0.0.1", "Redis host")
		cmd.Flags().IntVarP(&redisPort, "port", "P", 6379, "Redis port")
		cmd.Flags().StringVar(&redisPassword, "password", "", "Redis password")
		cmd.Flags().IntVar(&redisDB, "db", 0, "Redis database number")
	}
}

func runRedisConnect(_ *cobra.Command, _ []string) error {
	client, err := redisdb.Connect(redisHost, redisPort, redisPassword, redisDB)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	return redisdb.NewREPL(client).Run()
}

func runRedisExec(_ *cobra.Command, args []string) error {
	client, err := redisdb.Connect(redisHost, redisPort, redisPassword, redisDB)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	// Join all args as a single command string, then split respecting quotes.
	line := strings.Join(args, " ")
	parts := redisdb.SplitArgs(line)
	if len(parts) == 0 {
		return nil
	}
	iargs := make([]interface{}, len(parts))
	for i, p := range parts {
		iargs[i] = p
	}
	result, err := client.Do(context.Background(), iargs...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(result)
	return nil
}
