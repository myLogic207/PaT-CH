package api

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/mylogic207/PaT-CH/storage/cache"
	"github.com/mylogic207/PaT-CH/system"
)

type ApiConfig struct {
	Host       string       `json:"host"`
	Port       uint16       `json:"port"`
	PortOffset uint16       `json:"port_offset"`
	InitFile   string       `json:"init_file"`
	cert       *Certificate `json:"-"`
	Redis      bool         `json:"redis"`
	RedisConf  *cache.RedisConfig
}

func DefaultConfig() *ApiConfig {
	return &ApiConfig{
		Host:       "localhost",
		Port:       80,
		PortOffset: 0,
		InitFile:   "",
		Redis:      false,
		cert:       nil,
	}
}

func (c *ApiConfig) Addr() string {
	return c.Host + ":" + fmt.Sprint(c.Port+c.PortOffset)
}

func (c *ApiConfig) init() error {
	if c.Host == "" {
		c.Host = ""
		logger.Println("could not determine host, using all")
	}
	if c.Port == 0 {
		c.Port = 80
		logger.Println("could not determine port, using default")
	}
	if c.PortOffset == 0 {
		logger.Println("could not determine port offset, using none")
	}
	if c.InitFile == "" {
		logger.Println("could not determine init file, using none")
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
	if c.cert != nil {
		if c.cert.Cert == "" {
			return errors.New("certificate file not specified")
		}
		if c.cert.Key == "" {
			return errors.New("key file not specified")
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
		if p, err := strconv.ParseUint(val, 10, 16); err != nil {
			logger.Println("could not parse 'port', using default")
			config.Port = 80
		} else {
			config.Port = uint16(p)
		}
	}

	if val, ok := toConf.Get("portoffset"); ok {
		if p, err := strconv.ParseUint(val, 10, 16); err != nil {
			logger.Println("could not parse 'portoffset', using none")
			config.PortOffset = 0
		} else {
			config.PortOffset = uint16(p)
		}
	}

	if val, ok := toConf.Get("initfile"); ok {
		config.InitFile = val
	}

	config.cert = &Certificate{}

	if val, ok := toConf.Get("cert"); ok {
		config.cert.Cert = val
	}

	if val, ok := toConf.Get("key"); ok {
		config.cert.Key = val
	}

	if config.cert.Key == "" && config.cert.Cert == "" {
		logger.Println("no certificate provided, not using tls")
		config.cert = nil
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
