package cmd

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/ZacharyJo/db-cli/internal/config"
	"github.com/ZacharyJo/db-cli/internal/db"
	"github.com/ZacharyJo/db-cli/internal/importer"
)

var (
	importBatchSize    int
	importStopOnError  bool
	importVerbose      bool
	importCreateDB     bool
	importOnConflict   string
	importIgnoreErrors bool
	importGaussCompat  string
)

var importCmd = &cobra.Command{
	Use:   "import <file.sql>",
	Short: "Import and execute a SQL file",
	Args:  cobra.ExactArgs(1),
	Example: `  db-cli import --type mysql -H 127.0.0.1 -u root -p secret -d mydb ./dump.sql
  db-cli import --type mysql -H 127.0.0.1 -u root -p secret -d newdb ./dump.sql --create-db
  db-cli import --type oceanbase -H 10.0.0.1 -u app -p secret -d mydb ./schema.sql --stop-on-error
  db-cli import --type gaussdb -H 10.0.0.1 -u root -p secret -d mydb ./dump.sql --gaussdb-compat M --create-db --ignore-errors`,
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
	importCmd.Flags().BoolVar(&importCreateDB, "create-db", false,
		"create the target database if it does not exist")
	importCmd.Flags().StringVar(&importOnConflict, "on-conflict", "",
		"conflict resolution for INSERT: ignore (rewrites INSERT to skip duplicate keys)")
	importCmd.Flags().BoolVar(&importIgnoreErrors, "ignore-errors", false,
		"skip all errors and continue importing (errors are still reported at the end)")
	importCmd.Flags().StringVar(&importGaussCompat, "gaussdb-compat", "",
		"DBCOMPATIBILITY mode for GaussDB: 'M' enables MySQL-compat parsing and skips SQL rewriting; used with --create-db to create the database in that mode")
}

func runImport(_ *cobra.Command, args []string) error {
	sqlFile := args[0]

	// Propagate --gaussdb-compat into the config so EnsureDatabase picks it up.
	if importGaussCompat != "" {
		rootCfg.GaussDBCompatMode = importGaussCompat
	}

	if importCreateDB {
		if rootCfg.Database == "" {
			return fmt.Errorf("--create-db requires -d <database>")
		}
		if err := db.EnsureDatabase(rootCfg); err != nil {
			return err
		}
	}

	conn, err := db.Connect(rootCfg)
	if err != nil {
		return err
	}
	defer conn.Close()

	onConflict := importer.OnConflict(importOnConflict)
	// GaussDB MySQL-compat mode supports neither INSERT IGNORE nor ON CONFLICT DO NOTHING.
	// Suppress INSERT rewriting; --ignore-errors handles duplicate-key failures.
	if rootCfg.DBType == config.DBGaussDB {
		onConflict = importer.OnConflictDefault
	}

	opts := importer.Options{
		BatchSize:    importBatchSize,
		StopOnError:  importStopOnError,
		Verbose:      importVerbose,
		OnConflict:   onConflict,
		IgnoreErrors: importIgnoreErrors,
		// When GaussDB is in MySQL-compat mode, pass raw MySQL syntax through without rewriting.
		MysqlCompat: rootCfg.DBType == config.DBGaussDB && importGaussCompat == "M",
		// KingbaseDB and GaussDB both use the PostgreSQL wire protocol.
		PgWire:   rootCfg.DBType == config.DBKingbase || rootCfg.DBType == config.DBGaussDB,
		CreateDB: importCreateDB,
		Cfg:      rootCfg,
		OpenDB: func(cfg *config.Config) (*sql.DB, error) {
			return db.OpenSingleDB(cfg)
		},
		EnsureDB: func(cfg *config.Config) error {
			return db.EnsureDatabase(cfg)
		},
	}

	count, errs := importer.Import(conn.WriteDB(), sqlFile, opts)
	fmt.Printf("Imported %d statement(s).\n", count)
	if len(errs) > 0 {
		fmt.Fprintf(os.Stderr, "%d error(s) occurred:\n", len(errs))
		for _, e := range errs {
			fmt.Fprintf(os.Stderr, "  - %v\n", e)
		}
		return fmt.Errorf("%d import error(s)", len(errs))
	}
	return nil
}
