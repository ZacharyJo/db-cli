# Changelog

All notable changes to this project will be documented in this file.

## [2.2.1] - 2026-05-10

### Added
- `--version` / `-v` flag: print the binary version (injected at build time via ldflags)

### Fixed
- Security: escape SQL identifiers in `--create-db` and `\c` to prevent injection via database names containing special characters (backtick for MySQL wire, double-quote for PostgreSQL wire)
- Security: apply `pgxQuote` to `user` and `dbname` fields in PostgreSQL-wire DSN (was previously only applied to `password`)
- Security: use `url.UserPassword` in MongoDB URI builder to correctly percent-encode credentials with special characters
- Security: percent-escape `user` and `schema` fields in Dameng DSN (was previously only applied to `password`)
- Security: validate `--gaussdb-compat` value — only alphanumeric characters and underscores are accepted
- `import --stop-on-error`: now correctly aborts on statement execution errors; both error branches in `flushStmt` were previously identical (dead code), causing `fatalErr` to never be set
- REPL: semicolon detection changed from `strings.Contains` to `strings.HasSuffix` to avoid false triggers on semicolons inside string literals (e.g. `VALUES('a;b')`)
- Port defaults for KingbaseDB / GaussDB / Dameng: use cobra `Changed("port")` instead of comparing against the MySQL default `3306`, so explicitly passing `-P 3306` is now respected
- History files unified under `~/.db-cli/` with per-type filenames (`mysql_history`, `gaussdb_history`, `redis_history`, `mongo_history`, etc.)
- `connect`, `exec`, `import`: replace `os.Exit(1)` inside `RunE` handlers with `return err` for correct cobra error propagation and non-zero exit code
- `import`: suppress cobra usage output on runtime errors (`SilenceUsage = true`) — avoids printing the help block when `--ignore-errors` reports accumulated errors at the end
- Table output: datetime values (e.g. Dameng driver returns `time.Time`) now render as `2006-01-02 15:04:05` instead of splitting across multiple rows; embedded CR/LF in cell values normalized to spaces; tablewriter auto word-wrap disabled
- `import --ignore-errors` on PostgreSQL-wire (GaussDB / KingbaseDB / openGauss): execute `ROLLBACK` after each failed statement to reset the aborted-transaction state, preventing cascading errors on subsequent statements

---

## [2.2.0] - 2026-05-09

### Added
- **GaussDB MySQL-compat mode**: `--gaussdb-compat M` flag for `import` command — creates the target database with `DBCOMPATIBILITY='M'` and passes raw MySQL SQL through without any client-side rewriting (backticks, `ENGINE=InnoDB`, `AUTO_INCREMENT`, `INSERT IGNORE`, etc. sent as-is). Use with `--create-db` to auto-create an M-mode database in one step.
- **openGauss native driver**: replaced pgx with the official `openGauss-connector-go-pq` driver for GaussDB connections, supporting SHA256, MD5SHA256, and SM3 authentication natively.
- **Single-arch cross-compilation**: `make build-target GOOS=... GOARCH=...` for targeted cross-compilation without building all platforms.

---

## [2.1.0] - 2026-04-30

### Added
- **Redis Sentinel support**: `--mode sentinel --addrs host:port,... --master-name NAME` for primary/replica failover clusters
- **Redis Cluster support**: `--mode cluster --addrs host:port,...` for sharded clusters
- `--addrs` flag for Redis: specify multiple node addresses (overrides `-H`/`-P`); `-H`/`-P` remain backward-compatible for single mode
- `import`: follow `USE dbname` statements and switch the active connection to the new database mid-import
- `import --create-db`: now also auto-creates databases encountered in `USE dbname` statements

### Fixed
- Redis: `\c N` (SELECT) now works correctly by rebuilding the client instead of reusing pool state
- Redis: `formatValue` handles `map[string]interface{}` responses (RESP3 protocol)
- Redis: `--mode` flag validates value; unsupported modes print a clear error
- importer: connection pool leak when `USE dbname` reconnect fails during `acquireConn` (newDB is now closed and state is restored)
- importer: `StopOnError` is now honoured after a failed `USE dbname` switch
- importer: verbose `SKIP>` log when `USE dbname` is encountered but no reconnect is configured

---

## [2.0.0] - 2026-04-30

### Breaking Changes
- Binary renamed from `mysql-cli` / `mysqlcli` to `db-cli`
- Go module path changed to `github.com/ZacharyJo/db-cli`
- Config file renamed from `~/.mysqlcli.toml` to `~/.db-cli.toml`
- Environment variable prefix changed from `MYSQLCLI_` to `DB_CLI_`

### Added
- **Dameng (DM8)** database support via MySQL compatibility mode (`--type dameng`), default port 5236
- **Redis** interactive REPL (`db-cli redis`) and one-shot execution (`db-cli redis exec`)
  - Meta-commands: `\d` (INFO keyspace), `\c N` (SELECT N), `\q`, `\h`
  - History file: `~/.db-cli_redis_history`
- **MongoDB** interactive REPL (`db-cli mongo`) and one-shot execution (`db-cli mongo exec`)
  - JSON runCommand input format with multi-line brace-balance detection
  - Meta-commands: `\d` (list databases), `\dt` (list collections), `\c DBNAME`, `\q`, `\h`
  - History file: `~/.db-cli_mongo_history`
- `\dt` meta-command: list tables in current database
  - MySQL wire: `SHOW TABLES`
  - PostgreSQL wire: queries `pg_tables` showing both schema and table name
- `\dn` meta-command: list schemas (PostgreSQL wire only; gaussdb, kingbase)
- `\c [DBNAME]` meta-command: switch database
  - PostgreSQL wire: reconnects (connections are bound to a specific database)
  - MySQL wire: executes `USE \`dbname\``
- `import --create-db`: create target database if it does not exist
  - MySQL wire: `CREATE DATABASE IF NOT EXISTS`
  - PostgreSQL wire: probes bootstrap DB (postgres → kingbase → template1)
- `import --on-conflict=ignore`: rewrite statements to skip conflicts
  - MySQL wire: `INSERT INTO` → `INSERT IGNORE INTO`
  - PostgreSQL wire: appends `ON CONFLICT DO NOTHING` to INSERT
  - Also rewrites `CREATE TABLE` / `CREATE INDEX` to `IF NOT EXISTS` variant
- `import --ignore-errors`: continue on any execution error (errors reported at end)
- Chinese documentation (`README_CN.md`)

### Fixed
- `import`: skip psql/ksql client meta-commands (`\c`, `\connect`, `\i`, etc.) that are not valid SQL
- `import`: skip `CREATE DATABASE` statements in dump files (target DB already selected at connect time)
- `import`: use a single dedicated `sql.Conn` for the entire import so `SET search_path` and other session-level state persists across all statements
- `import --on-conflict=ignore`: `INSERT IGNORE` rewrite is now idempotent (does not double-add `IGNORE`)
- `import --on-conflict=ignore`: `INSERT ... ON CONFLICT DO NOTHING` rewrite is now idempotent

### Changed
- `\d` now consistently uses `SHOW DATABASES` for all MySQL-wire databases

---

## [1.0.0] - 2026-04-30

### Added
- Initial release: `db-cli connect`, `db-cli exec`, `db-cli import`
- Supported databases: MySQL, MariaDB, OceanBase (MySQL wire); GaussDB, KingbaseDB (PostgreSQL wire)
- Read/write split: master + N slaves, round-robin read routing
- TLS/SSL support: disable / require / verify-ca / verify-full
- Output formats: table, JSON, CSV (switchable at runtime with `\output`)
- Named profiles via `~/.db-cli.toml`
- Environment variable configuration (`DB_CLI_*`)
- Cross-platform builds: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64
- Interactive REPL with readline history, multi-line input, `\e` editor integration
- Streaming SQL file import with `DELIMITER` directive support (stored procedures)
- `\timing` toggle for query timing
