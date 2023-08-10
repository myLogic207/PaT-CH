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

	if err := config.MergeDefault(redis_default_config); err != nil {
		logger.Println("could not parse redis config, using default(s)")
		config = util.NewConfig(redis_default_config, nil)
	}
	redisOptions := &redis.Options{}
	host, _ := config.Get("host")
	port, _ := config.Get("port")
	redisOptions.Addr = fmt.Sprintf("%s:%s", host, port)
	password, _ := config.GetString("password")
	redisOptions.Password = password
	db, _ := config.Get("db")
	if db, ok := db.(int); ok {
		redisOptions.DB = db
	} else {
		redisOptions.DB = 0
	}
	idleConn, _ := config.Get("idle.Conn")
	if idleConn, ok := idleConn.(int); ok {
		redisOptions.MaxIdleConns = idleConn
	} else {
		redisOptions.MaxIdleConns = 10
	}
	idleTimeout, _ := config.Get("Idle.Timeout")
	if idleTimeout, ok := idleTimeout.(int); ok {
		redisOptions.ConnMaxIdleTime = time.Duration(idleTimeout) * time.Second
	} else {
		redisOptions.ConnMaxIdleTime = 0
	}
	maxActive, _ := config.Get("pool")
	if maxActive, ok := maxActive.(int); ok {
		redisOptions.PoolSize = maxActive
	} else {
		redisOptions.PoolSize = 10
	}

	connection := redis.NewClient(redisOptions)
	status := connection.Ping(context.Background())
	if status.Err() != nil {
		logger.Println(status.Err())
		return nil, errors.New("could not connect to redis")
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
