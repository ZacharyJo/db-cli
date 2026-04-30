package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/ZacharyJo/mysql-cli-go/internal/db"
	"github.com/ZacharyJo/mysql-cli-go/internal/importer"
)

var (
	importBatchSize   int
	importStopOnError bool
	importVerbose     bool
)

var importCmd = &cobra.Command{
	Use:   "import <file.sql>",
	Short: "Import and execute a SQL file",
	Args:  cobra.ExactArgs(1),
	Example: `  mysqlcli import --type mysql -H 127.0.0.1 -u root -p secret -d mydb ./dump.sql
  mysqlcli import --type oceanbase -H 10.0.0.1 -u app -p secret -d mydb ./schema.sql --stop-on-error`,
	RunE: runImport,
}

func init() {
	rootCmd.AddCommand(importCmd)
	importCmd.Flags().IntVar(&importBatchSize, "batch-size", 0,
		"commit every N statements (0 = no explicit batching)")
	importCmd.Flags().BoolVar(&importStopOnError, "stop-on-error", false,
		"abort import on first error")
	importCmd.Flags().BoolVarP(&importVerbose, "verbose", "v", false,
		"print each statement before executing")
}

func runImport(_ *cobra.Command, args []string) error {
	sqlFile := args[0]

	conn, err := db.Connect(rootCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	opts := importer.Options{
		BatchSize:   importBatchSize,
		StopOnError: importStopOnError,
		Verbose:     importVerbose,
	}

	count, errs := importer.Import(conn.WriteDB(), sqlFile, opts)
	fmt.Printf("Imported %d statement(s).\n", count)
	if len(errs) > 0 {
		fmt.Fprintf(os.Stderr, "%d error(s) occurred:\n", len(errs))
		for _, e := range errs {
			fmt.Fprintf(os.Stderr, "  - %v\n", e)
		}
		os.Exit(1)
	}
	return nil
}
