package db

// gaussdb.go registers the openGauss driver.
// openGauss uses a custom SHA256 auth mechanism incompatible with pgx;
// the official connector handles it natively.

import (
	_ "gitee.com/opengauss/openGauss-connector-go-pq" // registers "opengauss" driver name
)
