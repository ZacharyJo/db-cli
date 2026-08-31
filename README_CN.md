# db-cli

[![CI](https://github.com/ZacharyJo/db-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/ZacharyJo/db-cli/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Release](https://img.shields.io/github/v/release/ZacharyJo/db-cli)](https://github.com/ZacharyJo/db-cli/releases)

统一的数据库命令行工具，支持 SQL 和 NoSQL 多种数据库。

[English Documentation](README.md)

**支持的数据库：**

| 数据库 | 协议 | 子命令 |
|--------|------|--------|
| MySQL / MariaDB | MySQL wire | `connect` / `exec` / `import` |
| OceanBase | MySQL wire | `connect` / `exec` / `import` |
| 达梦（DM8） | DM wire（`-t dameng`） | `connect` / `exec` / `import` |
| GaussDB / openGauss | openGauss 原生（`-t gaussdb`） | `connect` / `exec` / `import` |
| KingbaseDB | PostgreSQL wire（`-t kingbase`） | `connect` / `exec` / `import` |
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
cd db-cli
make build          # 输出至 bin/db-cli
```

**跨平台二进制：**

```bash
make build-all      # linux/amd64、linux/arm64、darwin/amd64、darwin/arm64、windows/amd64
```

## 快速开始

### SQL 数据库 — 交互式 REPL（`connect`）

```bash
db-cli connect --type mysql    -H 127.0.0.1 -P 3306  -u root   -p secret -d mydb
db-cli connect --type dameng   -H 127.0.0.1          -u SYSDBA  -p secret -d SYSDBA
db-cli connect --type gaussdb  -H 10.0.0.1           -u root   -p secret -d postgres
db-cli connect --type kingbase -H 10.0.0.1           -u system -p secret -d mydb
db-cli connect --profile prod-cluster
```

输入 SQL 语句，以 `;` 结尾后执行，支持多行输入。

**元命令：**

| 命令 | 说明 |
|------|------|
| `\q`、`\quit` | 退出 |
| `\h`、`\help` | 显示帮助 |
| `\d` | 列出所有数据库；达梦：列出所有用户/schema |
| `\dt` | 列出当前库的所有表；达梦：列出当前用户下的所有表 |
| `\dn` | 列出所有 Schema（PG wire：gaussdb、kingbase；达梦：同 `\d`） |
| `\c [NAME]` | 切换数据库；达梦：切换 schema（`SET SCHEMA`） |
| `\timing` | 开关查询计时 |
| `\output FORMAT` | 设置输出格式：`table` \| `json` \| `csv` |
| `\e` | 用 `$EDITOR` 编辑 SQL |
| `exit`、`quit` | `\q` 的别名 |

### SQL 数据库 — 一次性执行（`exec`）

```bash
db-cli exec --type mysql    -H 127.0.0.1 -u root  -p secret "SELECT version()"
db-cli exec --type dameng   -H 127.0.0.1 -P 5236  -u SYSDBA -p secret "SELECT * FROM V\$VERSION"
db-cli exec --type gaussdb  -H 10.0.0.1           -u root   -p secret "SELECT current_database()"
db-cli exec --type kingbase -H 10.0.0.1           -u system -p secret "SELECT version()"
```

### SQL 文件导入（`import`）

```bash
db-cli import --type mysql     -H 127.0.0.1 -u root   -p secret -d mydb   ./dump.sql
db-cli import --type dameng    -H 127.0.0.1 -P 5236   -u SYSDBA -p secret -d SYSDBA  ./schema.sql
db-cli import --type oceanbase -H 10.0.0.1  -u app    -p secret -d mydb   ./schema.sql --stop-on-error
db-cli import --type gaussdb   -H 10.0.0.1            -u root   -p secret -d mydb   ./dump.sql --gaussdb-compat M --create-db --ignore-errors
db-cli import --type kingbase  -H 10.0.0.1            -u system -p secret -d mydb   ./dump.sql --create-db
```

**导入参数：**

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--batch-size N` | 0 | 每 N 条语句提交一次事务（0 表示不分批） |
| `--on-conflict` | — | `ignore`：将 `INSERT` 改写为 `INSERT IGNORE`（MySQL）或追加 `ON CONFLICT DO NOTHING`（PG）；同时为 `CREATE TABLE/INDEX` 添加 `IF NOT EXISTS` |
| `--ignore-errors` | false | 忽略错误继续执行，最终汇总报告所有错误 |
| `--create-db` | false | 目标数据库不存在时自动创建 |
| `--stop-on-error` | false | 遇到错误立即终止 |
| `--verbose` / `-v` | false | 执行前打印每条语句 |
| `--gaussdb-compat M` | — | 仅限 GaussDB/openGauss：以 `DBCOMPATIBILITY='M'`（MySQL 兼容模式）创建数据库，并将原始 MySQL 语法直接透传给服务器，不做任何客户端改写。反引号、`ENGINE=InnoDB`、`AUTO_INCREMENT`、`INSERT IGNORE` 等均原样发送。配合 `--create-db` 可一步完成建库+导入。 |

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

#### 哨兵模式（主从）

```bash
db-cli redis --mode sentinel \
  --addrs 10.0.0.1:26379,10.0.0.2:26379,10.0.0.3:26379 \
  --master-name mymaster \
  --password secret
```

#### 集群模式

```bash
db-cli redis --mode cluster \
  --addrs 10.0.0.1:7000,10.0.0.2:7001,10.0.0.3:7002 \
  --password secret
```

> **注意：** `--db` 和 `\c`（SELECT）仅在 `single` 模式下可用。`sentinel` 模式支持 `--db`。`cluster` 模式下所有 key 共享单一逻辑 keyspace，不支持 SELECT。

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
| `-P`、`--port` | `3306` | 端口（gaussdb 自动切换为 `8000`，kingbase 自动切换为 `54321`，达梦自动切换为 `5236`） |
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
password = "SYSDBA001"
database = "SYSDBA"

[gaussdb-dev]
type     = "gaussdb"
host     = "127.0.0.1"
port     = 8000
user     = "dev"
password = "dev"
database = "devdb"
```

```bash
db-cli connect --profile prod-cluster
db-cli exec    --profile gaussdb-dev "SELECT version()"
```

## 达梦（DM8）使用说明

达梦采用 Oracle 风格的 schema 模型，与 MySQL/PostgreSQL 有以下差异：

### Schema 与用户

达梦没有独立的"数据库"概念，用户名即 schema 名。连接时 `-d` 参数填写用户名（schema）：

```bash
db-cli connect -t dameng -H 127.0.0.1 -P 5236 -u SYSDBA -p secret -d SYSDBA
```

### 元命令行为

| 命令 | 行为 |
|------|------|
| `\d` | 列出所有用户（`DBA_USERS`） |
| `\dt` | 列出当前用户下的所有表（`ALL_TABLES WHERE OWNER=USER`） |
| `\dn` | 同 `\d` |
| `\c NAME` | 切换到指定 schema（`SET SCHEMA "NAME"`） |

### 常用系统查询

```sql
-- 查看版本
SELECT * FROM V$VERSION;

-- 查看当前用户
SELECT USER FROM DUAL;

-- 查看所有表
SELECT OWNER, TABLE_NAME FROM ALL_TABLES WHERE OWNER = USER ORDER BY TABLE_NAME;

-- 查看表结构
SELECT COLUMN_NAME, DATA_TYPE, DATA_LENGTH, NULLABLE
FROM ALL_TAB_COLUMNS WHERE TABLE_NAME = 'YOUR_TABLE' AND OWNER = USER;
```

## GaussDB / openGauss 使用说明

GaussDB 使用私有的 SHA256 认证机制，与标准 pgx 驱动不兼容。db-cli 使用官方 [openGauss-connector-go-pq](https://gitee.com/opengauss/openGauss-connector-go-pq) 驱动，原生支持 SHA256、MD5SHA256 和 SM3 认证，同时兼容 **openGauss 社区版**和**华为云 GaussDB**。

默认端口：`8000`（未指定 `-P` 时自动切换）。

```bash
db-cli connect -t gaussdb -H 10.0.0.1 -P 8000 -u root -p secret -d postgres
db-cli connect --profile gaussdb-prod
```

### 数据库兼容模式

GaussDB 在建库时可指定兼容模式：

| 模式 | `DBCOMPATIBILITY` | SQL 方言 | 适用场景 |
|------|-------------------|----------|---------|
| M 模式 | `'M'` | MySQL 语法（反引号、`ENGINE=InnoDB`、`AUTO_INCREMENT`、`INSERT IGNORE`、`tinyint` 等） | `--gaussdb-compat M` |
| A 模式 | `'A'`（默认） | 标准 SQL / Oracle 风格；空字符串 `''` 被视为 `NULL` | 标准导入 |

**导入 MySQL dump 到 GaussDB（M 模式，推荐）：**

```bash
# 自动以 M 模式建库，原始 MySQL SQL 无需任何改写直接导入
db-cli import -t gaussdb -H 10.0.0.1 -u root -p secret -d mydb \
  --gaussdb-compat M --create-db --ignore-errors ./dump.sql
```

**导入 PostgreSQL 风格 SQL 到 GaussDB（A 模式）：**

文件需预先转换：`DATETIME` → `timestamp`、删除 `ON UPDATE CURRENT_TIMESTAMP`、`tinyint` → `smallint`、`longtext` → `text`、删除 `unsigned`、`CREATE SCHEMA IF NOT EXISTS` 去掉 `IF NOT EXISTS`、可能包含空字符串的列去掉 `NOT NULL`。

```bash
db-cli import -t gaussdb -H 10.0.0.1 -u root -p secret -d mydb \
  --create-db --ignore-errors ./dump_a_mode.sql
```

### 元命令行为

| 命令 | 行为 |
|------|------|
| `\d` | 列出所有数据库（`SELECT datname FROM pg_database`） |
| `\dt` | 列出当前 schema 下的所有表（`pg_tables WHERE schemaname = 'public'`） |
| `\dn` | 列出所有 Schema（`SELECT nspname FROM pg_namespace`） |
| `\c NAME` | 重新连接到指定数据库 |

### 常用系统查询

```sql
-- 查看版本
SELECT version();

-- 查看当前数据库
SELECT current_database();

-- 列出所有 schema
SELECT nspname FROM pg_namespace ORDER BY nspname;

-- 查看表结构
SELECT column_name, data_type, is_nullable
FROM information_schema.columns
WHERE table_name = 'your_table' AND table_schema = 'public';
```

## KingbaseDB 使用说明

KingbaseDB（ES 系列）使用 PostgreSQL wire 协议，与 pgx 驱动兼容。

默认端口：`54321`（未指定 `-P` 时自动切换）。

```bash
db-cli connect -t kingbase -H 10.0.0.1 -P 54321 -u system -p secret -d mydb
db-cli connect --profile kingbase-prod
```

### 元命令行为

| 命令 | 行为 |
|------|------|
| `\d` | 列出所有数据库 |
| `\dt` | 列出当前 schema 下的所有表 |
| `\dn` | 列出所有 schema |
| `\c NAME` | 重新连接到指定数据库 |

### 配置文件示例

```toml
[kingbase-prod]
type     = "kingbase"
host     = "10.61.120.31"
port     = 8342
user     = "system"
password = "secret"
database = "mydb"
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
