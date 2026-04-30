# db-cli

统一的数据库命令行工具，支持 SQL 和 NoSQL 多种数据库。

[English Documentation](README.md)

**支持的数据库：**

| 数据库 | 协议 | 子命令 |
|--------|------|--------|
| MySQL / MariaDB | MySQL wire | `connect` / `exec` / `import` |
| OceanBase | MySQL wire | `connect` / `exec` / `import` |
| 达梦（DM8） | MySQL wire | `connect` / `exec` / `import` |
| GaussDB / openGauss | PostgreSQL wire | `connect` / `exec` / `import` |
| KingbaseDB | PostgreSQL wire | `connect` / `exec` / `import` |
| Redis | Redis 协议 | `redis` / `redis exec` |
| MongoDB | MongoDB 协议 | `mongo` / `mongo exec` |

## 安装

**从源码安装：**

```bash
go install github.com/ZacharyJo/db-cli@latest
```

**本地构建：**

```bash
git clone https://github.com/ZacharyJo/db-cli
cd mysql-cli-go
make build          # 输出至 bin/db-cli
```

**跨平台二进制：**

```bash
make build-all      # linux/amd64、linux/arm64、darwin/amd64、darwin/arm64、windows/amd64
```

## 快速开始

### SQL 数据库 — 交互式 REPL（`connect`）

```bash
db-cli connect --type mysql    -H 127.0.0.1 -P 3306 -u root   -p secret -d mydb
db-cli connect --type dameng   -H 127.0.0.1         -u SYSDBA  -p secret -d SYSDBA
db-cli connect --type gaussdb  -H 10.0.0.1  -P 5432 -u admin  -p secret -d mydb
db-cli connect --type kingbase -H 10.0.0.1  -P 5432 -u system -p secret -d mydb
db-cli connect --profile prod-cluster
```

输入 SQL 语句，以 `;` 结尾后执行，支持多行输入。

**元命令：**

| 命令 | 说明 |
|------|------|
| `\q`、`\quit` | 退出 |
| `\h`、`\help` | 显示帮助 |
| `\d` | 列出所有数据库 |
| `\dt` | 列出当前库的所有表 |
| `\dn` | 列出所有 Schema（仅 PostgreSQL wire：gaussdb、kingbase） |
| `\c [DBNAME]` | 切换到指定数据库（不带参数则显示当前库名） |
| `\timing` | 开关查询计时 |
| `\output FORMAT` | 设置输出格式：`table` \| `json` \| `csv` |
| `\e` | 用 `$EDITOR` 编辑 SQL |
| `exit`、`quit` | `\q` 的别名 |

### SQL 数据库 — 一次性执行（`exec`）

```bash
db-cli exec --type mysql   -H 127.0.0.1 -u root  -p secret "SELECT version()"
db-cli exec --type gaussdb -H 10.0.0.1  -u admin -p secret "SELECT current_database()"
```

### SQL 文件导入（`import`）

```bash
db-cli import --type mysql     -H 127.0.0.1 -u root -p secret -d mydb ./dump.sql
db-cli import --type oceanbase -H 10.0.0.1  -u app  -p secret -d mydb ./schema.sql --stop-on-error
```

**导入参数：**

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--batch-size N` | 0 | 每 N 条语句提交一次事务（0 表示不分批） |
| `--stop-on-error` | false | 遇到错误立即终止 |
| `--verbose` / `-v` | false | 执行前打印每条语句 |

导入器逐行流式读取文件（不全量加载到内存），正确处理引号字符串、`--` 和 `/* */` 注释，以及 MySQL 的 `DELIMITER` 指令（存储过程场景）。

### Redis — 交互式 REPL

```bash
db-cli redis -H 127.0.0.1 -P 6379
db-cli redis -H 10.0.0.1  -P 6379 --password secret --db 1
```

直接输入 Redis 命令（如 `GET mykey`、`SET foo bar`、`HGETALL myhash`）即可执行。

**元命令：**

| 命令 | 说明 |
|------|------|
| `\q`、`\quit` | 退出 |
| `\h`、`\help` | 显示帮助 |
| `\d` | 显示键空间信息（`INFO keyspace`） |
| `\c N` | 切换到 Redis 数据库 N（`SELECT N`） |

### Redis — 一次性执行

```bash
db-cli redis exec -H 127.0.0.1 "GET mykey"
db-cli redis exec -H 127.0.0.1 SET foo bar
```

### MongoDB — 交互式 REPL

```bash
db-cli mongo -H 127.0.0.1 -P 27017 -d mydb
db-cli mongo -H 10.0.0.1  -u admin --password secret -d mydb
```

以 JSON 格式输入 MongoDB 命令（`runCommand` 风格）：

```json
{"find": "users", "filter": {"age": {"$gt": 18}}, "limit": 10}
{"insert": "logs", "documents": [{"msg": "hello"}]}
{"drop": "oldcollection"}
```

支持多行输入，直到大括号平衡后自动执行。

**元命令：**

| 命令 | 说明 |
|------|------|
| `\q`、`\quit` | 退出 |
| `\h`、`\help` | 显示帮助 |
| `\d` | 列出所有数据库 |
| `\dt` | 列出当前库的所有集合 |
| `\c [DBNAME]` | 切换到指定数据库（不带参数则显示当前库名） |

### MongoDB — 一次性执行

```bash
db-cli mongo exec -H 127.0.0.1 -d mydb '{"find":"users","filter":{},"limit":5}'
db-cli mongo exec -H 127.0.0.1 -d mydb '{"dbStats":1}'
```

## 全局参数（SQL 子命令）

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `-t`、`--type` | `mysql` | 数据库类型：`mysql` \| `oceanbase` \| `dameng` \| `gaussdb` \| `kingbase` |
| `-H`、`--host` | `127.0.0.1` | 主机地址 |
| `-P`、`--port` | `3306` | 端口（PG wire 类型自动切换为 `5432`，达梦自动切换为 `5236`） |
| `-u`、`--user` | | 用户名 |
| `-p`、`--password` | | 密码 |
| `-d`、`--database` | | 数据库名 |
| `-o`、`--output` | `table` | 输出格式：`table` \| `json` \| `csv` |
| `--profile` | | 配置文件中的命名 profile |
| `--config` | | 配置文件路径（默认：`~/.db-cli.toml`） |

## 读写分离

集群场景下，指定主库和从库：

```bash
db-cli connect --master 10.0.0.1:3306 --slaves 10.0.0.2:3306,10.0.0.3:3306 \
  -u app -p secret -d mydb
```

- `SELECT`、`SHOW`、`DESCRIBE`/`DESC`、`EXPLAIN` 查询路由到从库（轮询）。
- 其他所有语句路由到主库。
- 未配置从库时回退到主库。

## TLS / SSL

| 参数 | 说明 |
|------|------|
| `--ssl-mode` | `disable` \| `require` \| `verify-ca` \| `verify-full` |
| `--ssl-ca` | CA 证书 PEM 路径 |
| `--ssl-cert` | 客户端证书 PEM 路径 |
| `--ssl-key` | 客户端私钥 PEM 路径 |

```bash
db-cli connect --type mysql -H db.example.com -u root -p secret \
  --ssl-mode verify-full --ssl-ca /etc/ssl/ca.pem \
  --ssl-cert /etc/ssl/client-cert.pem --ssl-key /etc/ssl/client-key.pem
```

## 配置文件（`~/.db-cli.toml`）

通过命名 profile 避免重复输入连接参数：

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
password = "SYSDBA"
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

## 环境变量

所有参数均可通过 `DB_CLI_*` 环境变量设置：

```bash
export DB_CLI_HOST=10.0.0.1
export DB_CLI_USER=app
export DB_CLI_PASSWORD=secret
db-cli connect --type mysql -d mydb
```

优先级：命令行参数 > 环境变量 > 配置文件 profile。

## 开发

```bash
make build      # 构建当前平台二进制 → bin/db-cli
make build-all  # 跨平台编译
make test       # go test ./... -v
make fmt        # gofmt -w .
make vet        # go vet ./...
make clean      # 清理 bin/
```

单独测试某个包：

```bash
go test ./internal/config/... -v
go test ./internal/importer/... -v -run TestSplitQuotedSemicolon
```

## License

MIT
