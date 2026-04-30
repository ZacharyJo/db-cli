package redisdb

import (
	"context"
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"
)

// Mode 表示 Redis 连接模式。
type Mode string

const (
	ModeSingle   Mode = "single"   // 单机（默认）
	ModeSentinel Mode = "sentinel" // 主从哨兵
	ModeCluster  Mode = "cluster"  // 集群
)

// Options 包含连接所需的全部参数。
type Options struct {
	Addrs      []string // 单机: ["host:port"]；哨兵/集群: 多个节点
	Password   string
	DB         int    // 仅单机/哨兵有效；集群忽略
	MasterName string // 哨兵模式必填（sentinel master name）
	Mode       Mode   // single | sentinel | cluster
}

// Client 封装 go-redis UniversalClient。
type Client struct {
	rdb  redis.UniversalClient
	opts Options
}

// Connect 根据 Options.Mode 创建对应的 Redis 客户端并 PING 验证。
func Connect(opts Options) (*Client, error) {
	if len(opts.Addrs) == 0 {
		opts.Addrs = []string{"127.0.0.1:6379"}
	}
	if opts.Mode == "" {
		opts.Mode = ModeSingle
	}

	masterName := opts.MasterName
	if opts.Mode != ModeSentinel {
		masterName = "" // UniversalClient uses non-empty MasterName to detect sentinel mode
	}
	uopts := &redis.UniversalOptions{
		Addrs:      opts.Addrs,
		Password:   opts.Password,
		DB:         opts.DB,
		MasterName: masterName,
	}

	rdb := redis.NewUniversalClient(uopts)
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		rdb.Close()
		return nil, fmt.Errorf("redis connect %s: %w", strings.Join(opts.Addrs, ","), err)
	}
	return &Client{rdb: rdb, opts: opts}, nil
}

// Close 关闭底层连接。
func (c *Client) Close() error {
	return c.rdb.Close()
}

// Do 执行原始 Redis 命令，返回可读字符串。
func (c *Client) Do(ctx context.Context, args ...interface{}) (string, error) {
	val, err := c.rdb.Do(ctx, args...).Result()
	if err == redis.Nil {
		return "(nil)", nil
	}
	if err != nil {
		return "", err
	}
	return formatValue(val), nil
}

// SelectDB 切换 DB 编号（仅单机模式有效）。
// 通过重建客户端实现，避免连接池中 SELECT 状态不一致的问题。
func (c *Client) SelectDB(ctx context.Context, db int) error {
	if c.opts.Mode != ModeSingle {
		return fmt.Errorf("SELECT is only supported in single mode (current: %s)", c.opts.Mode)
	}
	newOpts := c.opts
	newOpts.DB = db
	selectMasterName := newOpts.MasterName
	if newOpts.Mode != ModeSentinel {
		selectMasterName = ""
	}
	uopts := &redis.UniversalOptions{
		Addrs:      newOpts.Addrs,
		Password:   newOpts.Password,
		DB:         newOpts.DB,
		MasterName: selectMasterName,
	}
	newRDB := redis.NewUniversalClient(uopts)
	if err := newRDB.Ping(ctx).Err(); err != nil {
		newRDB.Close()
		return fmt.Errorf("reconnect for SELECT %d: %w", db, err)
	}
	c.rdb.Close()
	c.rdb = newRDB
	c.opts.DB = db
	return nil
}

// Keyspace 返回 INFO keyspace 内容。
func (c *Client) Keyspace(ctx context.Context) (string, error) {
	return c.rdb.Info(ctx, "keyspace").Result()
}

// AddrString 返回连接地址的可读字符串（用于 prompt）。
func (c *Client) AddrString() string {
	return strings.Join(c.opts.Addrs, ",")
}

// CurrentDB 返回当前 DB 编号（集群/哨兵返回 0）。
func (c *Client) CurrentDB() int {
	return c.opts.DB
}

// CurrentMode 返回当前连接模式。
func (c *Client) CurrentMode() Mode {
	return c.opts.Mode
}

// formatValue 将 go-redis 返回值转为可打印字符串。
func formatValue(v interface{}) string {
	switch val := v.(type) {
	case nil:
		return "(nil)"
	case string:
		return val
	case int64:
		return fmt.Sprintf("(integer) %d", val)
	case []interface{}:
		if len(val) == 0 {
			return "(empty array)"
		}
		parts := make([]string, len(val))
		for i, item := range val {
			parts[i] = fmt.Sprintf("%d) %s", i+1, formatValue(item))
		}
		return strings.Join(parts, "\n")
	case map[interface{}]interface{}:
		if len(val) == 0 {
			return "(empty map)"
		}
		parts := make([]string, 0, len(val)*2)
		for k, v := range val {
			parts = append(parts, fmt.Sprintf("%s => %s", formatValue(k), formatValue(v)))
		}
		return strings.Join(parts, "\n")
	case map[string]interface{}:
		if len(val) == 0 {
			return "(empty map)"
		}
		parts := make([]string, 0, len(val)*2)
		for k, v := range val {
			parts = append(parts, fmt.Sprintf("%s => %s", k, formatValue(v)))
		}
		return strings.Join(parts, "\n")
	default:
		return fmt.Sprintf("%v", val)
	}
}
