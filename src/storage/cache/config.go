package cache

import (
	"fmt"
	"strconv"

	"github.com/mylogic207/PaT-CH/system"
)

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

func DefaultConfig() *RedisConfig {
	return &RedisConfig{
		Host:       "localhost",
		Port:       6379,
		Password:   "",
		DB:         1,
		Idle:       10,
		MaxActive:  100,
		TimeoutSec: 60,
	}
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

	if val, ok := config.Get("maxactive"); ok {
		p, err := strconv.ParseInt(val, 10, 32)
		if err != nil {
			logger.Println("could not parse redis max_active, using default")
		} else {
			outConf.MaxActive = int(p)
		}
	} else {
		logger.Println("could not determine redis max_active, using default")
	}

	if val, ok := config.Get("idletimeout"); ok {
		p, err := strconv.ParseInt(val, 10, 32)
		if err != nil {
			logger.Println("could not parse redis idle_timeout, using default")
		} else {
			outConf.TimeoutSec = int(p)
		}
	} else {
		logger.Println("could not determine redis idle_timeout, using default")
	}

	return outConf, err
}
