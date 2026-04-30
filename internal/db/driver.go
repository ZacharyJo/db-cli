package db

import (
	"database/sql"
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/ZacharyJo/db-cli/internal/config"
)

// Connector holds the master and slave connection pools.
type Connector struct {
	Master *sql.DB
	Slaves []*sql.DB
	cfg    *config.Config
	robin  uint64 // atomic counter for round-robin slave selection
}

// OpenSingleDB opens a single *sql.DB to the master host in cfg.
// Intended for use cases that need a standalone connection (e.g. switching
// databases during import) without the full Connector read/write split.
func OpenSingleDB(cfg *config.Config) (*sql.DB, error) {
	return openDB(cfg, cfg.EffectiveHost())
}

// Connect creates a Connector for the given Config.
// It opens a master connection and all slave connections lazily verified with Ping.
func Connect(cfg *config.Config) (*Connector, error) {
	c := &Connector{cfg: cfg}

	masterDB, err := openDB(cfg, cfg.EffectiveHost())
	if err != nil {
		return nil, fmt.Errorf("connect master %s: %w", cfg.EffectiveHost(), err)
	}
	masterDB.SetMaxOpenConns(25)
	masterDB.SetMaxIdleConns(5)
	c.Master = masterDB

	for _, addr := range cfg.Slaves {
		slaveDB, err := openDB(cfg, addr)
		if err != nil {
			// Non-fatal: skip unreachable slaves, warn but continue.
			fmt.Printf("warning: slave %s unavailable: %v\n", addr, err)
			continue
		}
		slaveDB.SetMaxOpenConns(10)
		slaveDB.SetMaxIdleConns(3)
		c.Slaves = append(c.Slaves, slaveDB)
	}
	return c, nil
}

// openDB opens a single *sql.DB connection for the given host address.
// The address format for MySQL is "host:port"; for PG wire it is "host:port" too
// (split internally when building the DSN).
func openDB(cfg *config.Config, addr string) (*sql.DB, error) {
	var driverName, dsn string

	switch {
	case cfg.IsDamengNative():
		driverName = "dm"
		dsn = BuildDamengDSN(cfg, addr)

	case cfg.IsMySQLCompatible():
		driverName = "mysql"
		// Ensure TLS is registered before opening.
		if cfg.SSLMode != "disable" && cfg.SSLMode != "" {
			if err := RegisterMySQLTLS(cfg); err != nil {
				return nil, err
			}
		}
		dsn = config.BuildMySQLDSN(cfg, addr)

	case cfg.IsPgCompatible():
		driverName = "pgx"
		host, port := splitHostPort(addr, defaultPgPort(cfg))
		dsn = config.BuildPgxDSN(cfg, host, port)

	default:
		return nil, fmt.Errorf("unsupported db type %q", cfg.DBType)
	}

	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// QueryDB picks a slave for SELECT queries (round-robin) or falls back to master.
func (c *Connector) QueryDB() *sql.DB {
	if len(c.Slaves) == 0 {
		return c.Master
	}
	idx := atomic.AddUint64(&c.robin, 1) % uint64(len(c.Slaves))
	return c.Slaves[idx]
}

// WriteDB always returns the master connection.
func (c *Connector) WriteDB() *sql.DB {
	return c.Master
}

// Close closes master and all slave connections.
func (c *Connector) Close() {
	if c.Master != nil {
		c.Master.Close()
	}
	for _, s := range c.Slaves {
		s.Close()
	}
}

// IsReadQuery returns true if the SQL appears to be a read-only statement.
func IsReadQuery(query string) bool {
	trimmed := strings.TrimSpace(strings.ToUpper(query))
	for _, prefix := range []string{"SELECT", "SHOW", "DESCRIBE", "DESC", "EXPLAIN"} {
		if strings.HasPrefix(trimmed, prefix) {
			return true
		}
	}
	return false
}

// splitHostPort splits "host:port" — if no port is present, returns defaultPort.
func splitHostPort(addr, defaultPort string) (host, port string) {
	if idx := strings.LastIndex(addr, ":"); idx >= 0 {
		return addr[:idx], addr[idx+1:]
	}
	return addr, defaultPort
}

func defaultPgPort(cfg *config.Config) string {
	return fmt.Sprintf("%d", cfg.Port)
}
