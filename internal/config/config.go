package config

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// DBType constants
const (
	DBMySQL      = "mysql"
	DBOceanBase  = "oceanbase"
	DBGaussDB    = "gaussdb"
	DBKingbase   = "kingbase"
)

// Config holds all connection parameters.
type Config struct {
	DBType         string
	Host           string
	Port           int
	Master         string   // host:port for R/W split master
	Slaves         []string // host:port list for read replicas
	User           string
	Password       string
	Database       string
	SSLMode        string // disable | require | verify-ca | verify-full
	SSLCA          string // path to CA cert PEM
	SSLCert        string // path to client cert PEM
	SSLKey         string // path to client key PEM
	ConnectTimeout int    // seconds
	Profile        string
	OutputFormat   string // table | json | csv
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		DBType:         DBMySQL,
		Host:           "127.0.0.1",
		Port:           3306,
		SSLMode:        "disable",
		ConnectTimeout: 10,
		OutputFormat:   "table",
	}
}

// LoadProfile loads a named profile from the Viper config file into cfg,
// overriding only fields that are set in the profile.
func LoadProfile(cfg *Config, profile string) error {
	if profile == "" {
		return nil
	}
	key := strings.ToLower(profile)
	if !viper.IsSet(key) {
		return fmt.Errorf("profile %q not found in config", profile)
	}
	sub := viper.Sub(key)
	if sub == nil {
		return fmt.Errorf("profile %q is empty", profile)
	}
	if v := sub.GetString("type"); v != "" {
		cfg.DBType = v
	}
	if v := sub.GetString("host"); v != "" {
		cfg.Host = v
	}
	if v := sub.GetInt("port"); v != 0 {
		cfg.Port = v
	}
	if v := sub.GetString("master"); v != "" {
		cfg.Master = v
	}
	if v := sub.GetStringSlice("slaves"); len(v) > 0 {
		cfg.Slaves = v
	}
	if v := sub.GetString("user"); v != "" {
		cfg.User = v
	}
	if v := sub.GetString("password"); v != "" {
		cfg.Password = v
	}
	if v := sub.GetString("database"); v != "" {
		cfg.Database = v
	}
	if v := sub.GetString("ssl-mode"); v != "" {
		cfg.SSLMode = v
	}
	if v := sub.GetString("ssl-ca"); v != "" {
		cfg.SSLCA = v
	}
	if v := sub.GetString("ssl-cert"); v != "" {
		cfg.SSLCert = v
	}
	if v := sub.GetString("ssl-key"); v != "" {
		cfg.SSLKey = v
	}
	return nil
}

// BuildMySQLDSN constructs a go-sql-driver/mysql DSN string.
func BuildMySQLDSN(cfg *Config, host string) string {
	timeout := time.Duration(cfg.ConnectTimeout) * time.Second
	tlsPart := ""
	if cfg.SSLMode != "disable" && cfg.SSLMode != "" {
		tlsPart = "&tls=custom"
	}
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?timeout=%s&charset=utf8mb4&parseTime=true&loc=Local%s",
		cfg.User, cfg.Password, host, cfg.Database, timeout, tlsPart)
	return dsn
}

// BuildPgxDSN constructs a pgx (PostgreSQL-wire) DSN string for GaussDB/KingbaseDB.
func BuildPgxDSN(cfg *Config, host, port string) string {
	sslMode := cfg.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s connect_timeout=%d",
		host, port, cfg.User, cfg.Password, cfg.Database, sslMode, cfg.ConnectTimeout)
	if cfg.SSLCA != "" {
		dsn += " sslrootcert=" + cfg.SSLCA
	}
	if cfg.SSLCert != "" {
		dsn += " sslcert=" + cfg.SSLCert
	}
	if cfg.SSLKey != "" {
		dsn += " sslkey=" + cfg.SSLKey
	}
	return dsn
}

// BuildTLSConfig constructs a tls.Config from the cert paths.
func BuildTLSConfig(cfg *Config) (*tls.Config, error) {
	tlsCfg := &tls.Config{}
	if cfg.SSLCA != "" {
		caPEM, err := os.ReadFile(cfg.SSLCA)
		if err != nil {
			return nil, fmt.Errorf("read CA cert: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caPEM) {
			return nil, fmt.Errorf("failed to parse CA cert")
		}
		tlsCfg.RootCAs = pool
	}
	if cfg.SSLCert != "" && cfg.SSLKey != "" {
		cert, err := tls.LoadX509KeyPair(cfg.SSLCert, cfg.SSLKey)
		if err != nil {
			return nil, fmt.Errorf("load client cert: %w", err)
		}
		tlsCfg.Certificates = []tls.Certificate{cert}
	}
	switch cfg.SSLMode {
	case "require":
		tlsCfg.InsecureSkipVerify = true //nolint:gosec
	case "verify-ca", "verify-full":
		tlsCfg.InsecureSkipVerify = false
	}
	return tlsCfg, nil
}

// EffectiveHost returns the host:port string to connect to.
// If Master is set it takes precedence; otherwise Host:Port is used.
func (c *Config) EffectiveHost() string {
	if c.Master != "" {
		return c.Master
	}
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// IsMySQLCompatible returns true for MySQL-wire databases.
func (c *Config) IsMySQLCompatible() bool {
	return c.DBType == DBMySQL || c.DBType == DBOceanBase
}

// IsPgCompatible returns true for PostgreSQL-wire databases.
func (c *Config) IsPgCompatible() bool {
	return c.DBType == DBGaussDB || c.DBType == DBKingbase
}
