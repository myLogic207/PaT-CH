package cache

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/myLogic207/PaT-CH/pkg/util"
	"github.com/redis/go-redis/v9"
)

type RedisConnector struct {
	active bool
	store  *redis.Client
	logger *log.Logger
}

func NewStubConnector() *RedisConnector {
	return &RedisConnector{
		store:  nil,
		active: false,
		logger: nil,
	}
}

func NewConnector(config *util.ConfigMap, logger *log.Logger) (*RedisConnector, error) {
	if logger == nil {
		logger = log.Default()
	}
	redisConfig, err := ParseConf(config)
	if err != nil {
		logger.Println("could not parse redis config, using default(s)")
	}
	return NewConnectorWithConf(redisConfig, logger)
}

func NewConnectorWithConf(config *RedisConfig, logger *log.Logger) (*RedisConnector, error) {
	if logger == nil {
		logger = log.Default()
	}
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
		store:  connection,
		active: true,
	}, nil
}

func (c *RedisConnector) Close() error {
	c.active = false
	return c.store.Close()
}

func (c *RedisConnector) Get(ctx context.Context, key string) (interface{}, bool) {
	// if Redis get from Redis
	if !c.active {
		c.logger.Println(errors.New("redis not active"))
		return "", false
	}
	if val, err := c.store.Get(ctx, key).Result(); err != nil {
		c.logger.Println(err)
		return "", false
	} else {
		fmt.Println(val)
		return val, true
	}
}

func (c *RedisConnector) Set(ctx context.Context, key string, value interface{}) bool {
	if !c.active {
		c.logger.Println(errors.New("redis not active"))
		return false
	}
	if val, err := c.store.Set(ctx, key, value, 0).Result(); err != nil {
		c.logger.Println(err)
		return false
	} else {
		fmt.Println(val)
		return true
	}
}

func (c *RedisConnector) Delete(ctx context.Context, key string) bool {
	if !c.active {
		c.logger.Println(errors.New("redis not active"))
		return false
	}
	if val, err := c.store.Del(ctx, key).Result(); err != nil {
		c.logger.Println(err)
		return false
	} else {
		fmt.Println(val)
		return true
	}
}

func (c *RedisConnector) Is_active() bool {
	return c.active
}
