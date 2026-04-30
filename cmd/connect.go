package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/ZacharyJo/mysql-cli-go/internal/db"
	"github.com/ZacharyJo/mysql-cli-go/internal/repl"
)

var connectCmd = &cobra.Command{
	Use:   "connect",
	Short: "Open an interactive SQL session",
	Example: `  mysqlcli connect --type mysql -H 127.0.0.1 -P 3306 -u root -p secret -d mydb
  mysqlcli connect --type gaussdb -H 10.0.0.1 -P 5432 -u admin -p secret -d mydb
  mysqlcli connect --profile prod-cluster`,
	RunE: runConnect,
}

func init() {
	rootCmd.AddCommand(connectCmd)
}

func runConnect(_ *cobra.Command, _ []string) error {
	conn, err := db.Connect(rootCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	r := repl.New(conn, rootCfg.DBType, rootCfg.EffectiveHost(), rootCfg.Database, rootCfg.OutputFormat)
	return r.Run()
}
