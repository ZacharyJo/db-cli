package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/ZacharyJo/db-cli/internal/config"
)

var (
	cfgFile string
	rootCfg = config.DefaultConfig()
)

var rootCmd = &cobra.Command{
	Use:   "db-cli",
	Short: "A cross-database CLI tool supporting MySQL, OceanBase, GaussDB, KingbaseDB, Dameng, Redis, and MongoDB",
	Long: `db-cli is a unified CLI for connecting to and querying multiple database types:
  - MySQL / MariaDB
  - OceanBase (MySQL-compatible)
  - GaussDB / openGauss (PostgreSQL-wire)
  - KingbaseDB (PostgreSQL-wire)
  - Dameng DM8 (native DM protocol)
  - Redis (single / sentinel / cluster)
  - MongoDB

It supports interactive REPL, one-shot SQL execution, SQL file import,
read/write split for clusters, TLS/SSL, and cross-platform distribution.`,
}

// Execute is the entry point called from main.go.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Config file flag.
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: ~/.db-cli.toml)")

	// Connection flags.
	rootCmd.PersistentFlags().StringVarP(&rootCfg.DBType, "type", "t", "mysql",
		"database type: mysql | oceanbase | gaussdb | kingbase | dameng")
	rootCmd.PersistentFlags().StringVarP(&rootCfg.Host, "host", "H", "127.0.0.1", "host")
	rootCmd.PersistentFlags().IntVarP(&rootCfg.Port, "port", "P", 3306, "port")
	rootCmd.PersistentFlags().StringVarP(&rootCfg.User, "user", "u", "", "username")
	rootCmd.PersistentFlags().StringVarP(&rootCfg.Password, "password", "p", "", "password")
	rootCmd.PersistentFlags().StringVarP(&rootCfg.Database, "database", "d", "", "database name")
	rootCmd.PersistentFlags().StringVar(&rootCfg.Profile, "profile", "", "named profile from config file")

	// Cluster R/W split flags.
	rootCmd.PersistentFlags().StringVar(&rootCfg.Master, "master", "", "master host:port (R/W split)")
	rootCmd.PersistentFlags().StringSliceVar(&rootCfg.Slaves, "slaves", nil,
		"comma-separated slave host:port list (R/W split)")

	// TLS flags.
	rootCmd.PersistentFlags().StringVar(&rootCfg.SSLMode, "ssl-mode", "disable",
		"TLS mode: disable | require | verify-ca | verify-full")
	rootCmd.PersistentFlags().StringVar(&rootCfg.SSLCA, "ssl-ca", "", "path to CA certificate PEM")
	rootCmd.PersistentFlags().StringVar(&rootCfg.SSLCert, "ssl-cert", "", "path to client certificate PEM")
	rootCmd.PersistentFlags().StringVar(&rootCfg.SSLKey, "ssl-key", "", "path to client key PEM")

	// Output format flag.
	rootCmd.PersistentFlags().StringVarP(&rootCfg.OutputFormat, "output", "o", "table",
		"output format: table | json | csv")

	// Bind env vars (DB_CLI_HOST, DB_CLI_USER, etc.)
	viper.SetEnvPrefix("DB_CLI")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err == nil {
			viper.AddConfigPath(home)
		}
		viper.SetConfigName(".db-cli")
		viper.SetConfigType("toml")
	}

	if err := viper.ReadInConfig(); err != nil {
		// Config file is optional — silence "not found" errors.
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Fprintf(os.Stderr, "warning: config file error: %v\n", err)
		}
	}

	// Apply named profile if requested.
	if rootCfg.Profile != "" {
		if err := config.LoadProfile(rootCfg, rootCfg.Profile); err != nil {
			fmt.Fprintf(os.Stderr, "warning: %v\n", err)
		}
	}

	// Default port by DB type if user did not change it.
	if rootCfg.DBType == config.DBKingbase && rootCfg.Port == 3306 {
		rootCfg.Port = 54321
	}
	if rootCfg.DBType == config.DBGaussDB && rootCfg.Port == 3306 {
		rootCfg.Port = 8000
	}
	if rootCfg.DBType == config.DBDameng && rootCfg.Port == 3306 {
		rootCfg.Port = 5236
	}
}
