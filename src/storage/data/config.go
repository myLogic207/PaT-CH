package data

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/mylogic207/PaT-CH/storage/cache"
	"github.com/mylogic207/PaT-CH/system"
)

type DataConfig struct {
	// add json
	Host         string `json:"host"`
	Port         uint16 `json:"port"`
	User         string `json:"user"`
	password     string
	DBname       string             `json:"name"`
	SSLmode      string             `json:"sslmode"` // disable, require, verify-ca, verify-full
	MaxConns     int                `json:"maxconns"`
	ConnLifetime string             `json:"connlifetime"` // 1h, 1m, 1s
	UseCache     bool               `json:"usecache"`     // true, false
	RedisConf    *cache.RedisConfig `json:"redis"`
	InitFile     string             `json:"initfile"`
}

func DefaultConfig() *DataConfig {
	return &DataConfig{
		Host:         "localhost",
		Port:         5432,
		User:         "postgres",
		password:     "",
		DBname:       "patch_db",
		SSLmode:      "disable",
		MaxConns:     10,
		ConnLifetime: "1h",
		UseCache:     false,
		RedisConf:    nil,
	}
}

func (c *DataConfig) init() error {
	if c.Host == "" {
		c.Host = "localhost"
	}
	if c.Port == 0 {
		c.Port = 5432
	}
	if c.User == "" {
		c.User = "postgres"
	}
	if c.DBname == "" {
		c.DBname = "patch_db"
	}
	if c.SSLmode == "" {
		c.SSLmode = "disable"
	}
	if c.MaxConns == 0 {
		c.MaxConns = 10
	}
	if c.ConnLifetime == "" {
		c.ConnLifetime = "1h"
	}
	if c.UseCache {
		if c.RedisConf == nil {
			return errors.New("redis config is nil")
		}
	}
	return nil
}

func (c *DataConfig) ConnString() string {
	return fmt.Sprintf("postgresql://%s:%s@%s:%d/%s?sslmode=%s", c.User, c.password, c.Host, c.Port, c.DBname, c.SSLmode)
}

func parseConfig(rawConf *system.ConfigMap, rc *system.ConfigMap) (*DataConfig, error) {
	if rawConf == nil {
		return nil, errors.New("config is nil")
	}
	conf := DefaultConfig()
	if val, ok := rawConf.Get("host"); ok {
		conf.Host = val
	}
	if val, ok := rawConf.Get("port"); ok {
		if v, err := strconv.ParseUint(val, 10, 16); err != nil {
			return nil, err
		} else {
			conf.Port = uint16(v)
		}
	}
	if val, ok := rawConf.Get("user"); ok {
		conf.User = val
	}
	if val, ok := rawConf.Get("password"); ok {
		conf.password = val
	}
	if val, ok := rawConf.Get("name"); ok {
		conf.DBname = val
	}
	if val, ok := rawConf.Get("sslmode"); ok {
		conf.SSLmode = val
	}
	if val, ok := rawConf.Get("maxconns"); ok {
		if v, err := strconv.Atoi(val); err != nil {
			return nil, err
		} else {
			conf.MaxConns = v
		}
	}
	if val, ok := rawConf.Get("connlifetime"); ok {
		conf.ConnLifetime = val
	}
	var err error
	if conf.RedisConf, err = cache.ParseConf(rc); err != nil {
		return nil, err
	}
	return conf, nil
}
