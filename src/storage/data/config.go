package data

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/mylogic207/PaT-CH/system"
)

type DataConfig struct {
	host     string
	port     uint16
	user     string
	password string
	dbname   string
	sslmode  string
}

func DefaultConfig() *DataConfig {
	return &DataConfig{
		host:     "localhost",
		port:     5432,
		user:     "postgres",
		password: "",
		dbname:   "patch_db",
	}
}

func (c *DataConfig) init() error {
	if c.host == "" {
		c.host = "localhost"
	}
	if c.port == 0 {
		c.port = 5432
	}
	if c.user == "" {
		c.user = "postgres"
	}
	if c.dbname == "" {
		c.dbname = "patch_db"
	}
	return nil
}

func (c *DataConfig) ConnString() string {
	return fmt.Sprintf("postgresql://%s:%s@%s:%d/%s?sslmode=%s", c.user, c.password, c.host, c.port, c.dbname, c.sslmode)
}

func parseConfig(rawConf *system.ConfigMap) (*DataConfig, error) {
	if rawConf == nil {
		return nil, errors.New("config is nil")
	}
	conf := DefaultConfig()
	if val, ok := rawConf.Get("host"); ok {
		conf.host = val
	}
	if val, ok := rawConf.Get("port"); ok {
		if v, err := strconv.ParseUint(val, 10, 16); err != nil {
			return nil, err
		} else {
			conf.port = uint16(v)
		}
	}
	if val, ok := rawConf.Get("user"); ok {
		conf.user = val
	}
	if val, ok := rawConf.Get("password"); ok {
		conf.password = val
	}
	if val, ok := rawConf.Get("name"); ok {
		conf.dbname = val
	}
	if val, ok := rawConf.Get("sslmode"); ok {
		conf.sslmode = val
	}
	return conf, nil
}
