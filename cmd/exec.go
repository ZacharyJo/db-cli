package cmd

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/ZacharyJo/db-cli/internal/db"
	"github.com/ZacharyJo/db-cli/internal/output"
)

var execCmd = &cobra.Command{
	Use:   "exec <SQL>",
	Short: "Execute a SQL statement and print results",
	Args:  cobra.ExactArgs(1),
	Example: `  db-cli exec --type mysql -H 127.0.0.1 -u root -p secret "SELECT version()"
  db-cli exec --type gaussdb -H 10.0.0.1 -u admin -p secret "SELECT current_database()"`,
	RunE: runExec,
}

func init() {
	rootCmd.AddCommand(execCmd)
}

func runExec(_ *cobra.Command, args []string) error {
	sql := args[0]
	conn, err := db.Connect(rootCfg)
	if err != nil {
		return err
	}
	defer conn.Close()

	printer := output.New(rootCfg.OutputFormat)
	start := time.Now()

	if db.IsReadQuery(sql) {
		rows, err := conn.QueryDB().Query(sql)
		elapsed := time.Since(start)
		if err != nil {
			return err
		}
		defer rows.Close()
		if err := printer.PrintRows(rows, elapsed); err != nil {
			return err
		}
	} else {
		res, err := conn.WriteDB().Exec(sql)
		elapsed := time.Since(start)
		if err != nil {
			return err
		}
		printer.PrintResult(res, elapsed)
	}
	return nil
}
