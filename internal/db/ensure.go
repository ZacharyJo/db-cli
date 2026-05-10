package db

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/ZacharyJo/db-cli/internal/config"
)

// pgBootstrapCandidates are probed in order until one succeeds.
// Standard PostgreSQL uses "postgres"; KingbaseDB uses "kingbase" or "template1".
var pgBootstrapCandidates = []string{"postgres", "kingbase", "template1"}

// EnsureDatabase creates the target database if it does not exist.
// It connects without specifying a database, runs CREATE DATABASE, then returns.
// The caller is expected to call Connect(cfg) normally afterwards.
func EnsureDatabase(cfg *config.Config) error {
	if cfg.IsMySQLCompatible() {
		bootstrapCfg := *cfg
		bootstrapCfg.Database = ""
		db, err := openDB(&bootstrapCfg, bootstrapCfg.EffectiveHost())
		if err != nil {
			return fmt.Errorf("bootstrap connect: %w", err)
		}
		defer db.Close()
		return ensureMysql(db, cfg.Database)
	}
	if cfg.IsPgCompatible() {
		return ensurePgWithProbe(cfg)
	}
	return fmt.Errorf("unsupported db type %q for --create-db", cfg.DBType)
}

// ensurePgWithProbe tries each bootstrap candidate database in order.
func ensurePgWithProbe(cfg *config.Config) error {
	var lastErr error
	for _, candidate := range pgBootstrapCandidates {
		bootstrapCfg := *cfg
		bootstrapCfg.Database = candidate
		db, err := openDB(&bootstrapCfg, bootstrapCfg.EffectiveHost())
		if err != nil {
			lastErr = err
			continue
		}
		defer db.Close()
		return ensurePg(db, cfg.Database, cfg.GaussDBCompatMode)
	}
	return fmt.Errorf("bootstrap connect: none of %v reachable: %w", pgBootstrapCandidates, lastErr)
}

func ensureMysql(db *sql.DB, dbname string) error {
	_, err := db.Exec("CREATE DATABASE IF NOT EXISTS `" + strings.ReplaceAll(dbname, "`", "``") + "`")
	if err != nil {
		return fmt.Errorf("create database %q: %w", dbname, err)
	}
	fmt.Printf("Database %q created (or already exists).\n", dbname)
	return nil
}

func ensurePg(db *sql.DB, dbname string, compatMode string) error {
	if compatMode != "" {
		for _, c := range compatMode {
			if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_') {
				return fmt.Errorf("invalid --gaussdb-compat value %q: only letters, digits, and underscores allowed", compatMode)
			}
		}
	}
	var exists bool
	err := db.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", dbname,
	).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check database %q: %w", dbname, err)
	}
	if exists {
		fmt.Printf("Database %q already exists.\n", dbname)
		return nil
	}
	stmt := `CREATE DATABASE "` + strings.ReplaceAll(dbname, `"`, `""`) + `"`
	if compatMode != "" {
		stmt += ` DBCOMPATIBILITY '` + compatMode + `'`
	}
	if _, err := db.Exec(stmt); err != nil {
		return fmt.Errorf("create database %q: %w", dbname, err)
	}
	fmt.Printf("Database %q created.\n", dbname)
	return nil
}
