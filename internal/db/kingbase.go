package db

// kingbase.go — KingbaseDB uses the PostgreSQL wire protocol (ES series).
// It uses the pgx/v5 stdlib driver (registered separately from gaussdb's openGauss driver).
// This file is a placeholder for any KingbaseDB-specific quirks (e.g. schema
// search_path defaults) that may need to be applied after connection.

import (
	_ "github.com/jackc/pgx/v5/stdlib" // registers "pgx" driver name for KingbaseDB
)
