package cache

import (
	"context"
	"errors"
	"log"

	"github.com/mylogic207/PaT-CH/system"
	"github.com/redis/go-redis/v9"
)

var logger = log.New(log.Writer(), "cache: ", log.Flags())

type RedisConnector struct {
	Active bool
	Store  *redis.Client
	ctx    context.Context
}

func NewConnector(config *system.ConfigMap, ctx context.Context) (*RedisConnector, error) {
	redisConfig, err := ParseConf(config)
	if err != nil {
		logger.Println("could not parse redis config, using default(s)")
	}
	return NewConnectorWithConf(redisConfig, ctx)
}

func NewConnectorWithConf(config *RedisConfig, ctx context.Context) (*RedisConnector, error) {
	connection := redis.NewClient(&redis.Options{
		Addr:     config.Addr(),
		Password: config.Password,
		DB:       config.DB,
	})
	status := connection.Ping(ctx)
	if status.Err() != nil {
		logger.Println(status.Err())
		return nil, errors.New("could not connect to redis")
	}
	logger.Println(status.String())
	return &RedisConnector{
		ctx:   ctx,
		Store: connection,
	}, nil
}

func (c *RedisConnector) Close() error {
	return c.Store.Close()
}

func (c *RedisConnector) Get(key string, ctx context.Context) (string, error) {
	// if Redis get from Redis
	if c.Active {
		return c.Store.Get(ctx, key).Result()
	}
	return "", errors.New("redis not active")
}

func (c *RedisConnector) Set(key string, value string, ctx context.Context) error {
	if c.Active {
		return c.Store.Set(ctx, key, value, 0).Err()
	}
	return errors.New("redis not active")
}
