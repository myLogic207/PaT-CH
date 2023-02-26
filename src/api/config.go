package api

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/mylogic207/PaT-CH/storage/cache"
	"github.com/mylogic207/PaT-CH/system"
)

type ApiConfig struct {
	Host      string `json:"host"`
	Port      uint16 `json:"http_port"`
	SPort     uint16 `json:"https_port"` // only used if secure is true
	CertFile  string `json:"cert_file"`  // only used if secure is true
	KeyFile   string `json:"key_file"`   // only used if secure is true
	Redis     bool   `json:"redis"`
	RedisConf *cache.RedisConfig
}

func DefaultConfig() *ApiConfig {
	return &ApiConfig{
		Host:     "localhost",
		Port:     2080,
		SPort:    2443,
		CertFile: "",
		KeyFile:  "",
		Redis:    false,
	}
}

func (c *ApiConfig) Addr() string {
	return c.Host + ":" + fmt.Sprint(c.Port)
}

func (c *ApiConfig) init() error {
	if c.Host == "" {
		c.Host = "localhost"
		logger.Println("could not determine host, using localhost")
	}
	if c.Port == 0 {
		c.Port = 2080
		logger.Println("could not determine port, using default")
	}
	if c.SPort == 0 {
		logger.Println("could not determine https port, not using https")
	}
	if c.Redis {
		if c.RedisConf.Host == "" {
			return errors.New("redis host not specified")
		}
		if c.RedisConf.Port == 0 {
			return errors.New("redis port not specified")
		}
		if c.RedisConf.Password == "" {
			return errors.New("redis password not specified")
		}
	}
	return nil
}

func ParseConf(toConf *system.ConfigMap, rc *system.ConfigMap) (*ApiConfig, error) {
	config := &ApiConfig{}

	if val, ok := toConf.Get("host"); ok {
		config.Host = val
	}

	if val, ok := toConf.Get("port"); ok {
		p, _ := strconv.ParseUint(val, 10, 16)
		config.Port = uint16(p)
	}

	if val, ok := toConf.Get("httpsport"); ok {
		p, _ := strconv.ParseUint(val, 10, 16)
		config.SPort = uint16(p)
	}

	if val, ok := toConf.Get("certfile"); ok {
		config.CertFile = val
	}

	if val, ok := toConf.Get("keyfile"); ok {
		config.KeyFile = val
	}

	if val, ok := toConf.Get("redis"); ok && rc != nil {
		use, err := strconv.ParseBool(val)
		if err != nil {
			logger.Println("could not parse 'redis', not using")
		} else {
			config.Redis = use
			config.RedisConf, err = cache.ParseConf(rc)
			if err != nil {
				logger.Println("could not parse redis config, not using")
			}
		}
	} else if ok {
		logger.Println("redis config not provided, not using")
	}

	logger.Println("Api finished parsing config")
	return config, nil
}
