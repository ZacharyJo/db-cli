package db

import (
	"fmt"
	"net/url"

	_ "gitee.com/chunanyong/dm"
	"github.com/ZacharyJo/db-cli/internal/config"
)

// BuildDamengDSN constructs a DSN for the native DM protocol driver.
// Format: dm://user:password@host:port?schema=dbname
func BuildDamengDSN(cfg *config.Config, addr string) string {
	host, port := splitHostPort(addr, fmt.Sprintf("%d", cfg.Port))
	user := url.PathEscape(cfg.User)
	password := url.PathEscape(cfg.Password)
	schema := url.PathEscape(cfg.Database)
	dsn := fmt.Sprintf("dm://%s:%s@%s:%s?schema=%s&connectTimeout=%d000",
		user, password, host, port, schema, cfg.ConnectTimeout)
	if cfg.Password != password {
		dsn += "&escapeProcess=true"
	}
	return dsn
}
