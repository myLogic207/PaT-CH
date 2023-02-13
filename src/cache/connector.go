package cache

import (
	"context"
	"errors"
	"log"

	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

// type Cache interface {
// 	Get(context context.Context, key string) (string, error)
// 	Set(key string, value string) error
// }

type RedisConnector struct {
	db     int
	active bool
	Cache  *redis.Client
}

func NewConnector(url string, db int) *RedisConnector {
	options, err := redis.ParseURL(url)
	if err != nil {
		panic(err)
	}
	client := redis.NewClient(options)
	return &RedisConnector{
		db:     db,
		Cache:  client,
		active: false,
	}
}

func (c *RedisConnector) Connect() error {
	if err := c.Cache.Ping(ctx).Err(); err != nil {
		log.Println(err)
		c.active = false
		return nil
	}
	c.active = true
	log.Println("connected to Redis cache")
	return nil
}

func (c *RedisConnector) Close() error {
	if c.active {
		return c.Cache.Close()
	}
	return errors.New("redis already closed")
}

func (c *RedisConnector) Get(key string) (string, error) {
	// if Redis get from Redis
	if c.active {
		return c.Cache.Get(ctx, key).Result()
	}
	return "", errors.New("redis not active")
}

func (c *RedisConnector) Set(key string, value string) error {
	if c.active {
		return c.Cache.Set(ctx, key, value, 0).Err()
	}
	return errors.New("redis not active")
}
