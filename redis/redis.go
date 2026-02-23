package redis

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
)

// Config holds Redis configuration
type Config struct {
	Addr     string
	Password string
	DB       int
}

// DefaultConfig returns default Redis configuration
func DefaultConfig() *Config {
	return &Config{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	}
}

// Client wraps Redis client with application-specific methods
type Client struct {
	client *redis.Client
	ctx    context.Context
}

// Global Redis client instance
var globalClient *Client

// InitRedis initializes the Redis client
func InitRedis(config *Config) (*Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     config.Addr,
		Password: config.Password,
		DB:       config.DB,
	})

	ctx := context.Background()

	// Test connection
	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %v", err)
	}

	globalClient = &Client{
		client: client,
		ctx:    ctx,
	}

	return globalClient, nil
}

// GetClient returns the global Redis client
func GetClient() *Client {
	return globalClient
}

// Close closes the Redis connection
func (c *Client) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// Context returns the context for Redis operations
func (c *Client) Context() context.Context {
	return c.ctx
}

// InnerClient returns the underlying redis.Client for advanced operations
func (c *Client) InnerClient() *redis.Client {
	return c.client
}
