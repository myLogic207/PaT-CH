package cache

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

type Config struct {
	Host       string
	Port       uint16
	Password   string
	DB         int
	Idle       int
	MaxActive  int
	TimeoutSec int
	// Addr     func() string
}

func (c Config) Addr() string {
	return c.Host + ":" + fmt.Sprint(c.Port)
}

type RedisConnector struct {
	Active bool
	Store  *redis.Client
}

func NewConnector(config *Config) *RedisConnector {
	connection := redis.NewClient(&redis.Options{
		Addr:     config.Addr(),
		Password: config.Password,
		DB:       config.DB,
	})
	return &RedisConnector{
		Store:  connection,
		Active: false,
	}
}

func (c *RedisConnector) Connect() error {
	if err := c.Store.Ping(ctx).Err(); err != nil {
		log.Println(err)
		c.Active = false
		return nil
	}
	c.Active = true
	log.Println("connected to Redis cache")
	return nil
}

func (c *RedisConnector) Close() error {
	if c.Active {
		return c.Store.Close()
	}
	return errors.New("redis already closed")
}

func (c *RedisConnector) Get(key string) (string, error) {
	// if Redis get from Redis
	if c.Active {
		return c.Store.Get(ctx, key).Result()
	}
	return "", errors.New("redis not active")
}

func (c *RedisConnector) Set(key string, value string) error {
	if c.Active {
		return c.Store.Set(ctx, key, value, 0).Err()
	}
	return errors.New("redis not active")
}
