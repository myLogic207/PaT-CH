package cache

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/myLogic207/PaT-CH/pkg/util"
	"github.com/redis/go-redis/v9"
)

var (
	ErrMergeConfig     = errors.New("could not merge config")
	ErrCouldNotConnect = errors.New("could not connect to redis")
)

var redis_default_config = map[string]interface{}{
	"host":        "localhost",
	"port":        6379,
	"password":    "",
	"db":          0,
	"IdleTimeout": 0,
	"IdleConn":    10,
	"Pool":        10,
	"TimeoutSec":  60,
}

type RedisConnector struct {
	active bool
	config *util.Config
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

func NewConnector(config *util.Config, logger *log.Logger) (*RedisConnector, error) {
	if logger == nil {
		logger = log.Default()
	}

	if config == nil {
		config = util.NewConfig(redis_default_config, nil)
	} else {
		if err := config.MergeDefault(redis_default_config); err != nil {
			logger.Println(err)
			return nil, ErrMergeConfig
		}
	}

	redisOptions := &redis.Options{}
	host, _ := config.GetString("host")
	port, _ := config.GetString("port")
	redisOptions.Addr = fmt.Sprintf("%s:%s", host, port)
	password, _ := config.GetString("password")
	redisOptions.Password = password
	if db, ok := config.Get("db").(int); ok {
		redisOptions.DB = db
	} else {
		redisOptions.DB = 0
	}
	if idleConn, ok := config.Get("idle.Conn").(int); ok {
		redisOptions.MaxIdleConns = idleConn
	} else {
		redisOptions.MaxIdleConns = 10
	}
	if idleTimeout, ok := config.Get("Idle.Timeout").(int); ok {
		redisOptions.ConnMaxIdleTime = time.Duration(idleTimeout) * time.Second
	} else {
		redisOptions.ConnMaxIdleTime = 0
	}
	if maxActive, ok := config.Get("pool").(int); ok {
		redisOptions.PoolSize = maxActive
	} else {
		redisOptions.PoolSize = 10
	}

	connection := redis.NewClient(redisOptions)
	status := connection.Ping(context.Background())
	if status.Err() != nil {
		logger.Println(status.Err())
		return nil, ErrCouldNotConnect
	}
	logger.Println(status.String())
	return &RedisConnector{
		config: config,
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
