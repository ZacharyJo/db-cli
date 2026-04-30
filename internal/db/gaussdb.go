package db

// gaussdb.go registers the pgx stdlib driver.
// GaussDB (openGauss) uses the PostgreSQL wire protocol, so pgx works directly.
// The driver is imported via its side-effect registration in pgx.go.

import (
	_ "github.com/jackc/pgx/v5/stdlib" // registers "pgx" driver name
)
