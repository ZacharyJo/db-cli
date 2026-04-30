package redisdb

import (
	"context"
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"
)

// Client wraps a go-redis client with connection metadata.
type Client struct {
	rdb  *redis.Client
	Host string
	Port int
	DB   int
}

// Connect opens a Redis connection and verifies it with PING.
func Connect(host string, port int, password string, db int) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", host, port),
		Password: password,
		DB:       db,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		rdb.Close()
		return nil, fmt.Errorf("redis connect %s:%d: %w", host, port, err)
	}
	return &Client{rdb: rdb, Host: host, Port: port, DB: db}, nil
}

// Close closes the underlying connection.
func (c *Client) Close() error {
	return c.rdb.Close()
}

// Do executes a raw Redis command and returns a human-readable result string.
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

// SelectDB switches to a different Redis database number.
func (c *Client) SelectDB(ctx context.Context, db int) error {
	if err := c.rdb.Do(ctx, "SELECT", db).Err(); err != nil {
		return err
	}
	c.DB = db
	return nil
}

// Keyspace returns a summary of all non-empty databases from INFO keyspace.
func (c *Client) Keyspace(ctx context.Context) (string, error) {
	return c.rdb.Info(ctx, "keyspace").Result()
}

// formatValue converts go-redis result values to a printable string.
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
	default:
		return fmt.Sprintf("%v", val)
	}
}
