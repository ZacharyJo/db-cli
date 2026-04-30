# Changelog

All notable changes to this project will be documented in this file.

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
- History file renamed from `~/.db-cli_history` to `~/.db-cli_history` (SQL REPL)
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
