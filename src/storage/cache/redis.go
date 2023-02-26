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
	active bool
	store  *redis.Client
}

func NewConnector(config *system.ConfigMap) (*RedisConnector, error) {
	redisConfig, err := ParseConf(config)
	if err != nil {
		logger.Println("could not parse redis config, using default(s)")
	}
	return NewConnectorWithConf(redisConfig)
}

func NewConnectorWithConf(config *RedisConfig) (*RedisConnector, error) {
	connection := redis.NewClient(&redis.Options{
		Addr:     config.Addr(),
		Password: config.Password,
		DB:       config.DB,
	})
	status := connection.Ping(context.Background())
	if status.Err() != nil {
		logger.Println(status.Err())
		return nil, errors.New("could not connect to redis")
	}
	logger.Println(status.String())
	return &RedisConnector{
		store: connection,
	}, nil
}

func (c *RedisConnector) Close() error {
	return c.store.Close()
}

func (c *RedisConnector) Get(ctx context.Context, key string) (string, error) {
	// if Redis get from Redis
	if c.active {
		return c.store.Get(ctx, key).Result()
	}
	return "", errors.New("redis not active")
}

func (c *RedisConnector) Set(ctx context.Context, key string, value string) error {
	if c.active {
		return c.store.Set(ctx, key, value, 0).Err()
	}
	return errors.New("redis not active")
}
