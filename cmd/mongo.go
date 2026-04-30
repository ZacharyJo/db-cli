package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/ZacharyJo/db-cli/internal/mongodb"
)

var (
	mongoHost     string
	mongoPort     int
	mongoUser     string
	mongoPassword string
	mongoDB       string
)

var mongoCmd = &cobra.Command{
	Use:   "mongo",
	Short: "Connect to MongoDB and open an interactive session",
	Example: `  db-cli mongo -H 127.0.0.1 -P 27017 -d mydb
  db-cli mongo -H 10.0.0.1 -u admin --password secret -d mydb
  db-cli mongo exec -H 127.0.0.1 -d mydb '{"find":"users","filter":{}}'`,
	RunE: runMongoConnect,
}

var mongoExecCmd = &cobra.Command{
	Use:   "exec COMMAND",
	Short: "Execute a single MongoDB command (JSON) and print the result",
	Example: `  db-cli mongo exec -H 127.0.0.1 -d mydb '{"find":"users","filter":{},"limit":5}'
  db-cli mongo exec -H 127.0.0.1 -d mydb '{"dbStats":1}'`,
	Args: cobra.ExactArgs(1),
	RunE: runMongoExec,
}

func init() {
	rootCmd.AddCommand(mongoCmd)
	mongoCmd.AddCommand(mongoExecCmd)

	for _, cmd := range []*cobra.Command{mongoCmd, mongoExecCmd} {
		cmd.Flags().StringVarP(&mongoHost, "host", "H", "127.0.0.1", "MongoDB host")
		cmd.Flags().IntVarP(&mongoPort, "port", "P", 27017, "MongoDB port")
		cmd.Flags().StringVarP(&mongoUser, "user", "u", "", "MongoDB username")
		cmd.Flags().StringVar(&mongoPassword, "password", "", "MongoDB password")
		cmd.Flags().StringVarP(&mongoDB, "database", "d", "test", "MongoDB database name")
	}
}

func runMongoConnect(_ *cobra.Command, _ []string) error {
	client, err := mongodb.Connect(mongoHost, mongoPort, mongoUser, mongoPassword, mongoDB)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	return mongodb.NewREPL(client).Run()
}

func runMongoExec(_ *cobra.Command, args []string) error {
	client, err := mongodb.Connect(mongoHost, mongoPort, mongoUser, mongoPassword, mongoDB)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	result, err := client.RunCommand(context.Background(), args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(result)
	return nil
}
