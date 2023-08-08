package api

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/myLogic207/PaT-CH/pkg/storage/cache"
	"github.com/myLogic207/PaT-CH/pkg/util"
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
	}
	if c.Port == 0 {
		c.Port = 80
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

func ParseConf(toConf *util.ConfigMap, rc *util.ConfigMap) (*ApiConfig, error) {
	config := &ApiConfig{}

	if val, ok := toConf.Get("host"); ok {
		config.Host = val
	}

	if val, ok := toConf.Get("port"); ok {
		if p, err := strconv.ParseUint(val, 10, 16); err != nil {
			return nil, err
		} else {
			config.Port = uint16(p)
		}
	}

	if val, ok := toConf.Get("portoffset"); ok {
		if p, err := strconv.ParseUint(val, 10, 16); err != nil {
			return nil, err
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

	if val, ok := toConf.Get("redis"); ok && rc != nil {
		use, err := strconv.ParseBool(val)
		if err == nil {
			config.Redis = use
			config.RedisConf, err = cache.ParseConf(rc)
			if err != nil {
				return nil, err
			}
		}
	} else if ok && rc == nil {
		return nil, errors.New("redis config not provided, not using")
	}

	return config, nil
}
