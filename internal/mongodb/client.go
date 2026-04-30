package mongodb

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Client wraps a mongo.Client with connection metadata.
type Client struct {
	mc     *mongo.Client
	Host   string
	Port   int
	DBName string
}

// Connect opens a MongoDB connection and verifies it with Ping.
func Connect(host string, port int, user, password, dbName string) (*Client, error) {
	uri := buildURI(host, port, user, password)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mc, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("mongodb connect %s:%d: %w", host, port, err)
	}
	if err := mc.Ping(ctx, nil); err != nil {
		mc.Disconnect(context.Background()) //nolint:errcheck
		return nil, fmt.Errorf("mongodb ping %s:%d: %w", host, port, err)
	}
	return &Client{mc: mc, Host: host, Port: port, DBName: dbName}, nil
}

// Close disconnects the client.
func (c *Client) Close() {
	c.mc.Disconnect(context.Background()) //nolint:errcheck
}

// DB returns the currently selected database.
func (c *Client) DB() *mongo.Database {
	return c.mc.Database(c.DBName)
}

// UseDB switches the active database.
func (c *Client) UseDB(name string) {
	c.DBName = name
}

// ListDatabases returns all database names.
func (c *Client) ListDatabases(ctx context.Context) ([]string, error) {
	return c.mc.ListDatabaseNames(ctx, bson.D{})
}

// ListCollections returns all collection names in the current database.
func (c *Client) ListCollections(ctx context.Context) ([]string, error) {
	return c.DB().ListCollectionNames(ctx, bson.D{})
}

// RunCommand executes a raw command on the current database and returns the
// result as a formatted JSON-like string.
func (c *Client) RunCommand(ctx context.Context, cmdStr string) (string, error) {
	var doc bson.D
	if err := bson.UnmarshalExtJSON([]byte(cmdStr), false, &doc); err != nil {
		return "", fmt.Errorf("invalid command JSON %q: %w", cmdStr, err)
	}
	res := c.DB().RunCommand(ctx, doc)
	if res.Err() != nil {
		return "", res.Err()
	}
	var raw bson.M
	if err := res.Decode(&raw); err != nil {
		return "", err
	}
	out, err := bson.MarshalExtJSONIndent(raw, false, false, "", "  ")
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func buildURI(host string, port int, user, password string) string {
	if user != "" && password != "" {
		return fmt.Sprintf("mongodb://%s:%s@%s:%d", user, password, host, port)
	}
	if user != "" {
		return fmt.Sprintf("mongodb://%s@%s:%d", user, host, port)
	}
	return fmt.Sprintf("mongodb://%s:%d", host, port)
}
