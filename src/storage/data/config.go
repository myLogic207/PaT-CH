package data

import "fmt"

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
