# mysqlcli

A unified command-line tool for connecting to and querying multiple database types over MySQL or PostgreSQL wire protocols.

**Supported databases:**

| Database | Wire Protocol |
|----------|--------------|
| MySQL / MariaDB | MySQL |
| OceanBase | MySQL |
| GaussDB / openGauss | PostgreSQL |
| KingbaseDB | PostgreSQL |

## Installation

**From source:**

```bash
go install github.com/ZacharyJo/mysql-cli-go@latest
```

**Build locally:**

```bash
git clone https://github.com/ZacharyJo/mysql-cli-go
cd mysql-cli-go
make build          # outputs bin/mysqlcli
```

**Cross-platform binaries:**

```bash
make build-all      # linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64
```

## Quick Start

### Interactive REPL (`connect`)

```bash
mysqlcli connect --type mysql -H 127.0.0.1 -P 3306 -u root -p secret -d mydb
mysqlcli connect --type gaussdb -H 10.0.0.1 -P 5432 -u admin -p secret -d mydb
mysqlcli connect --profile prod-cluster
```

Once connected, type SQL and terminate with `;` to execute. Multi-line input is supported.

**Meta-commands:**

| Command | Description |
|---------|-------------|
| `\q`, `\quit` | Exit |
| `\h`, `\help` | Show help |
| `\d` | List databases |
| `\timing` | Toggle query timing |
| `\output FORMAT` | Set output format: `table` \| `json` \| `csv` |
| `\e` | Open `$EDITOR` to compose SQL |
| `exit`, `quit` | Alias for `\q` |

### One-shot Execution (`exec`)

```bash
mysqlcli exec --type mysql -H 127.0.0.1 -u root -p secret "SELECT version()"
mysqlcli exec --type gaussdb -H 10.0.0.1 -u admin -p secret "SELECT current_database()"
```

### SQL File Import (`import`)

```bash
mysqlcli import --type mysql -H 127.0.0.1 -u root -p secret -d mydb ./dump.sql
mysqlcli import --type oceanbase -H 10.0.0.1 -u app -p secret -d mydb ./schema.sql --stop-on-error
```

**Import flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--batch-size N` | 0 | Commit every N statements (0 = no batching) |
| `--stop-on-error` | false | Abort on first error |
| `--verbose` / `-v` | false | Print each statement before executing |

The importer streams the file line-by-line (no full load into memory) and correctly handles quoted strings, `--` and `/* */` comments, and MySQL's `DELIMITER` directive for stored procedures.

## Global Flags

These flags apply to all subcommands:

| Flag | Default | Description |
|------|---------|-------------|
| `-t`, `--type` | `mysql` | DB type: `mysql` \| `oceanbase` \| `gaussdb` \| `kingbase` |
| `-H`, `--host` | `127.0.0.1` | Host |
| `-P`, `--port` | `3306` | Port (auto-switches to `5432` for PG-wire types) |
| `-u`, `--user` | | Username |
| `-p`, `--password` | | Password |
| `-d`, `--database` | | Database name |
| `-o`, `--output` | `table` | Output format: `table` \| `json` \| `csv` |
| `--profile` | | Named profile from config file |
| `--config` | | Config file path (default: `~/.mysqlcli.toml`) |

## Read/Write Split

For cluster setups, specify a master and one or more read replicas:

```bash
mysqlcli connect --master 10.0.0.1:3306 --slaves 10.0.0.2:3306,10.0.0.3:3306 \
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
mysqlcli connect --type mysql -H db.example.com -u root -p secret \
  --ssl-mode verify-full --ssl-ca /etc/ssl/ca.pem \
  --ssl-cert /etc/ssl/client-cert.pem --ssl-key /etc/ssl/client-key.pem
```

## Config File (`~/.mysqlcli.toml`)

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

[dev]
type     = "gaussdb"
host     = "127.0.0.1"
port     = 5432
user     = "dev"
password = "dev"
database = "devdb"
```

```bash
mysqlcli connect --profile prod-cluster
mysqlcli exec --profile dev "SELECT version()"
```

## Environment Variables

All flags can be set via `MYSQLCLI_*` environment variables:

```bash
export MYSQLCLI_HOST=10.0.0.1
export MYSQLCLI_USER=app
export MYSQLCLI_PASSWORD=secret
mysqlcli connect --type mysql -d mydb
```

Priority: CLI flags > environment variables > config file profile.

## Development

```bash
make build      # build for current platform → bin/mysqlcli
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
