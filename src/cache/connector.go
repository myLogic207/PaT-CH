package cache

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"

	"github.com/mylogic207/PaT-CH/system"
	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

var logger = log.New(log.Writer(), "cache: ", log.Flags())

type RedisConfig struct {
	Host       string
	Port       uint16
	Password   string
	DB         int
	Idle       int
	MaxActive  int
	TimeoutSec int
	// Addr     func() string
}

func (c RedisConfig) Addr() string {
	return c.Host + ":" + fmt.Sprint(c.Port)
}

func ParseConf(config *system.ConfigMap) (*RedisConfig, error) {
	var err error
	outConf := &RedisConfig{
		Host:       "localhost",
		Port:       6379,
		Password:   "",
		DB:         1,
		Idle:       10,
		MaxActive:  100,
		TimeoutSec: 60,
	}
	if val, ok := config.Get("host"); ok {
		outConf.Host = val
	} else {
		logger.Println("could not determine redis host, using localhost")
	}
	if val, ok := config.Get("port"); ok {
		p, err := strconv.ParseUint(val, 10, 16)
		if err != nil {
			logger.Println("could not parse redis port, using default")
		} else {
			outConf.Port = uint16(p)
		}
	} else {
		logger.Println("could not determine redis port, using default")
	}

	if val, ok := config.Get("password"); ok {
		outConf.Password = val
	} else {
		logger.Println("could not determine redis password, using none")
	}

	if val, ok := config.Get("db"); ok {
		p, err := strconv.ParseInt(val, 10, 32)
		if err != nil {
			logger.Println("could not parse redis db, using default")
		} else {
			outConf.DB = int(p)
		}
	} else {
		logger.Println("could not determine redis db, using default")
	}

	if val, ok := config.Get("idle"); ok {
		p, err := strconv.ParseInt(val, 10, 32)
		if err != nil {
			logger.Println("could not parse redis idle, using default")
		} else {
			outConf.Idle = int(p)
		}
	} else {
		logger.Println("could not determine redis idle, using default")
	}

	if val, ok := config.Get("max_active"); ok {
		p, err := strconv.ParseInt(val, 10, 32)
		if err != nil {
			logger.Println("could not parse redis max_active, using default")
		} else {
			outConf.MaxActive = int(p)
		}
	} else {
		logger.Println("could not determine redis max_active, using default")
	}

	if val, ok := config.Get("timeout_sec"); ok {
		p, err := strconv.ParseInt(val, 10, 32)
		if err != nil {
			logger.Println("could not parse redis timeout_sec, using default")
		} else {
			outConf.TimeoutSec = int(p)
		}
	} else {
		logger.Println("could not determine redis timeout_sec, using default")
	}

	return outConf, err
}

type RedisConnector struct {
	Active bool
	Store  *redis.Client
}

func NewConnectorPreConf(config *system.ConfigMap) *RedisConnector {
	redisConfig, err := ParseConf(config)
	if err != nil {
		logger.Println("could not parse redis config, using default(s)")
	}
	return NewConnector(redisConfig)
}

func NewConnector(config *RedisConfig) *RedisConnector {
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
		logger.Println(err)
		c.Active = false
		return nil
	}
	c.Active = true
	logger.Println("connected to Redis cache")
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
