# db-cli

A unified command-line tool for connecting to and querying multiple databases — SQL and NoSQL alike.

[中文文档](README_CN.md)

**Supported databases:**

| Database | Protocol | Subcommand |
|----------|----------|------------|
| MySQL / MariaDB | MySQL wire | `connect` / `exec` / `import` |
| OceanBase | MySQL wire | `connect` / `exec` / `import` |
| Dameng (DM8) | DM wire (`-t dameng`) | `connect` / `exec` / `import` |
| GaussDB / openGauss | PostgreSQL wire | `connect` / `exec` / `import` |
| KingbaseDB | PostgreSQL wire | `connect` / `exec` / `import` |
| Redis | Redis protocol | `redis` / `redis exec` |
| MongoDB | MongoDB protocol | `mongo` / `mongo exec` |

## Installation

**From source:**

```bash
go install github.com/ZacharyJo/db-cli@latest
```

**Build locally:**

```bash
git clone https://github.com/ZacharyJo/db-cli
cd mysql-cli-go
make build          # outputs bin/db-cli
```

**Cross-platform binaries:**

```bash
make build-all      # linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64
```

## Quick Start

### SQL Databases — Interactive REPL (`connect`)

```bash
db-cli connect --type mysql    -H 127.0.0.1 -P 3306 -u root -p secret -d mydb
db-cli connect --type dameng   -H 127.0.0.1         -u SYSDBA -p secret -d SYSDBA
db-cli connect --type gaussdb  -H 10.0.0.1  -P 5432 -u admin -p secret -d mydb
db-cli connect --type kingbase -H 10.0.0.1  -P 5432 -u system -p secret -d mydb
db-cli connect --profile prod-cluster
```

Type SQL and terminate with `;` to execute. Multi-line input is supported.

**Meta-commands:**

| Command | Description |
|---------|-------------|
| `\q`, `\quit` | Exit |
| `\h`, `\help` | Show help |
| `\d` | List databases; dameng: list users/schemas |
| `\dt` | List tables in current database; dameng: tables owned by current user |
| `\dn` | List schemas (PG wire: gaussdb, kingbase; dameng: same as `\d`) |
| `\c [NAME]` | Switch database; dameng: switch schema (`SET SCHEMA`) |
| `\timing` | Toggle query timing |
| `\output FORMAT` | Set output format: `table` \| `json` \| `csv` |
| `\e` | Open `$EDITOR` to compose SQL |
| `exit`, `quit` | Alias for `\q` |

### SQL Databases — One-shot Execution (`exec`)

```bash
db-cli exec --type mysql   -H 127.0.0.1 -u root  -p secret "SELECT version()"
db-cli exec --type dameng  -H 127.0.0.1 -P 5236 -u SYSDBA -p secret "SELECT * FROM V\$VERSION"
db-cli exec --type gaussdb -H 10.0.0.1  -u admin -p secret "SELECT current_database()"
```

### SQL File Import (`import`)

```bash
db-cli import --type mysql     -H 127.0.0.1 -u root  -p secret -d mydb ./dump.sql
db-cli import --type dameng    -H 127.0.0.1 -P 5236 -u SYSDBA -p secret -d SYSDBA ./schema.sql
db-cli import --type oceanbase -H 10.0.0.1  -u app   -p secret -d mydb  ./schema.sql --stop-on-error
```

**Import flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--batch-size N` | 0 | Commit every N statements (0 = no batching) |
| `--stop-on-error` | false | Abort on first error |
| `--verbose` / `-v` | false | Print each statement before executing |

The importer streams the file line-by-line (no full load into memory) and correctly handles quoted strings, `--` and `/* */` comments, and MySQL's `DELIMITER` directive for stored procedures.

### Redis — Interactive REPL

```bash
db-cli redis -H 127.0.0.1 -P 6379
db-cli redis -H 10.0.0.1  -P 6379 --password secret --db 1
```

Enter Redis commands directly (e.g. `GET mykey`, `SET foo bar`, `HGETALL myhash`).

**Meta-commands:**

| Command | Description |
|---------|-------------|
| `\q`, `\quit` | Exit |
| `\h`, `\help` | Show help |
| `\d` | Show keyspace info (`INFO keyspace`) |
| `\c N` | Switch to Redis database N (`SELECT N`) |

#### Sentinel (Primary/Replica)

```bash
db-cli redis --mode sentinel \
  --addrs 10.0.0.1:26379,10.0.0.2:26379,10.0.0.3:26379 \
  --master-name mymaster \
  --password secret
```

#### Cluster

```bash
db-cli redis --mode cluster \
  --addrs 10.0.0.1:7000,10.0.0.2:7001,10.0.0.3:7002 \
  --password secret
```

> **Note:** `--db` and `\c` (SELECT) are only available in `single` mode. In `sentinel` mode, `--db` is supported. In `cluster` mode, all keys reside in a single logical keyspace.

### Redis — One-shot Execution

```bash
db-cli redis exec -H 127.0.0.1 "GET mykey"
db-cli redis exec -H 127.0.0.1 SET foo bar
```

### MongoDB — Interactive REPL

```bash
db-cli mongo -H 127.0.0.1 -P 27017 -d mydb
db-cli mongo -H 10.0.0.1  -u admin --password secret -d mydb
```

Enter MongoDB commands as JSON (`runCommand` format):

```json
{"find": "users", "filter": {"age": {"$gt": 18}}, "limit": 10}
{"insert": "logs", "documents": [{"msg": "hello"}]}
{"drop": "oldcollection"}
```

Multi-line input is supported — keep typing until braces are balanced.

**Meta-commands:**

| Command | Description |
|---------|-------------|
| `\q`, `\quit` | Exit |
| `\h`, `\help` | Show help |
| `\d` | List databases |
| `\dt` | List collections in current database |
| `\c [DBNAME]` | Switch to database (show current if omitted) |

### MongoDB — One-shot Execution

```bash
db-cli mongo exec -H 127.0.0.1 -d mydb '{"find":"users","filter":{},"limit":5}'
db-cli mongo exec -H 127.0.0.1 -d mydb '{"dbStats":1}'
```

## Global Flags (SQL subcommands)

| Flag | Default | Description |
|------|---------|-------------|
| `-t`, `--type` | `mysql` | DB type: `mysql` \| `oceanbase` \| `dameng` \| `gaussdb` \| `kingbase` |
| `-H`, `--host` | `127.0.0.1` | Host |
| `-P`, `--port` | `3306` | Port (auto-switches to `5432` for PG-wire types, `5236` for Dameng) |
| `-u`, `--user` | | Username |
| `-p`, `--password` | | Password |
| `-d`, `--database` | | Database name |
| `-o`, `--output` | `table` | Output format: `table` \| `json` \| `csv` |
| `--profile` | | Named profile from config file |
| `--config` | | Config file path (default: `~/.db-cli.toml`) |

## Read/Write Split

For cluster setups, specify a master and one or more read replicas:

```bash
db-cli connect --master 10.0.0.1:3306 --slaves 10.0.0.2:3306,10.0.0.3:3306 \
  -u app -p secret -d mydb
```

- `SELECT`, `SHOW`, `DESCRIBE`/`DESC`, `EXPLAIN` queries route to slaves (round-robin).
- All other statements route to master.
- Falls back to master when no slaves are configured.

## TLS / SSL

| Flag | Description |
|------|-------------|
| `--ssl-mode` | `disable` \| `require` \| `verify-ca` \| `verify-full` |
| `--ssl-ca` | Path to CA certificate PEM |
| `--ssl-cert` | Path to client certificate PEM |
| `--ssl-key` | Path to client key PEM |

```bash
db-cli connect --type mysql -H db.example.com -u root -p secret \
  --ssl-mode verify-full --ssl-ca /etc/ssl/ca.pem \
  --ssl-cert /etc/ssl/client-cert.pem --ssl-key /etc/ssl/client-key.pem
```

## Config File (`~/.db-cli.toml`)

Named profiles avoid repeating connection flags:

```toml
[prod-cluster]
type     = "mysql"
master   = "10.0.0.1:3306"
slaves   = ["10.0.0.2:3306", "10.0.0.3:3306"]
user     = "app"
password = "secret"
database = "mydb"
ssl-ca   = "/etc/ssl/ca.pem"

[dameng-dev]
type     = "dameng"
host     = "127.0.0.1"
port     = 5236
user     = "SYSDBA"
password = "SYSDBA001"
database = "SYSDBA"

[gaussdb-dev]
type     = "gaussdb"
host     = "127.0.0.1"
port     = 5432
user     = "dev"
password = "dev"
database = "devdb"
```

```bash
db-cli connect --profile prod-cluster
db-cli exec    --profile gaussdb-dev "SELECT version()"
```

## Dameng (DM8) Notes

Dameng uses an Oracle-style schema model. Key differences from MySQL/PostgreSQL:

### Schema and users

Dameng has no separate "database" concept — the username is the schema name. Use the username as the `-d` value:

```bash
db-cli connect -t dameng -H 127.0.0.1 -P 5236 -u SYSDBA -p secret -d SYSDBA
```

### Meta-command behavior

| Command | Behavior |
|---------|----------|
| `\d` | List all users (`DBA_USERS`) |
| `\dt` | List tables owned by current user (`ALL_TABLES WHERE OWNER=USER`) |
| `\dn` | Same as `\d` |
| `\c NAME` | Switch to the named schema (`SET SCHEMA "NAME"`) |

### Useful system queries

```sql
-- Version
SELECT * FROM V$VERSION;

-- Current user
SELECT USER FROM DUAL;

-- List tables
SELECT OWNER, TABLE_NAME FROM ALL_TABLES WHERE OWNER = USER ORDER BY TABLE_NAME;

-- Table structure
SELECT COLUMN_NAME, DATA_TYPE, DATA_LENGTH, NULLABLE
FROM ALL_TAB_COLUMNS WHERE TABLE_NAME = 'YOUR_TABLE' AND OWNER = USER;
```

## Environment Variables

All flags can be set via `DB_CLI_*` environment variables:

```bash
export DB_CLI_HOST=10.0.0.1
export DB_CLI_USER=app
export DB_CLI_PASSWORD=secret
db-cli connect --type mysql -d mydb
```

Priority: CLI flags > environment variables > config file profile.

## Development

```bash
make build      # build for current platform → bin/db-cli
make build-all  # cross-compile for all platforms
make test       # go test ./... -v
make fmt        # gofmt -w .
make vet        # go vet ./...
make clean      # remove bin/
```

Run a single package's tests:

```bash
go test ./internal/config/... -v
go test ./internal/importer/... -v -run TestSplitQuotedSemicolon
```

## License

MIT
