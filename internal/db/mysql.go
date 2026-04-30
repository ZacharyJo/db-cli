package db

import (
	"crypto/tls"

	"github.com/go-sql-driver/mysql"
	cfg "github.com/ZacharyJo/db-cli/internal/config"
)

// RegisterMySQLTLS registers a named TLS config with the MySQL driver.
// The DSN references it as "tls=custom".
func RegisterMySQLTLS(c *cfg.Config) error {
	tlsCfg, err := cfg.BuildTLSConfig(c)
	if err != nil {
		return err
	}
	if tlsCfg == nil {
		tlsCfg = &tls.Config{}
	}
	// RegisterTLSConfig is idempotent — re-registering the same name is fine.
	return mysql.RegisterTLSConfig("custom", tlsCfg)
}
