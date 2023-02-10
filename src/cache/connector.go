package cache

import (
	"context"
	"errors"

	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

// type Cache interface {
// 	Get(context context.Context, key string) (string, error)
// 	Set(key string, value string) error
// }

type Connector struct {
	RedisConnector
	FallBackConnector
}

type FallBackConnector struct {
	Cache map[string]string
}

type RedisConnector struct {
	active bool
	Cache  *redis.Client
}

func (c *Connector) Get(key string) (string, error) {
	// if Redis get from Redis
	if c.RedisConnector.active {
		return c.RedisConnector.Cache.Get(ctx, key).Result()
	}
	// Get from fallback
	if value, ok := c.FallBackConnector.Cache[key]; ok {
		return value, nil
	}
	return "", errors.New("key not found")
}

func (c *Connector) Set(key string, value string) error {
	if c.RedisConnector.active {
		return c.RedisConnector.Cache.Set(ctx, key, value, 0).Err()
	}
	c.FallBackConnector.Cache[key] = value
	return nil
}

func NewConnector(url string, db int) *Connector {
	return &Connector{
		RedisConnector:    NewRedisConnector(url, db),
		FallBackConnector: NewFallBackConnector(),
	}
}

func NewFallBackConnector() FallBackConnector {
	return FallBackConnector{
		Cache: make(map[string]string),
	}
}

func NewRedisConnector(url string, db int) RedisConnector {
	options, err := redis.ParseURL(url)
	if err != nil {
		panic(err)
	}
	client := redis.NewClient(options)
	return RedisConnector{
		Cache:  client,
		active: false,
	}
}

func (c *Connector) Connect() error {
	if err := c.RedisConnector.Cache.Ping(ctx).Err(); err != nil {
		println(err.Error())
		println("failed to connect to Redis cache, using in-memory fallback")
		c.RedisConnector.active = false
		return nil
	}
	c.RedisConnector.active = true
	return nil
}

func (c *Connector) Close() error {
	if c.RedisConnector.active {
		return c.RedisConnector.Cache.Close()
	}
	return nil
}
