package system

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
)

func twoSplit(variable string, splitter string) (string, string, error) {
	split := strings.Split(variable, splitter)
	if len(split) < 2 {
		return "", "", fmt.Errorf("cannot split into minimum (2) parts")
	}
	key, val := split[0], split[1:]
	return key, strings.Join(val, ""), nil
}

type Configurable interface {
	map[string]string
	Get(field string) (string, error)
	Set(field string, val string) error
}

type ConfigMap map[string]string

func (c ConfigMap) Get(field string) (string, bool) {
	if val := c[strings.ToLower(field)]; val != "" {
		return val, true
	}
	return "", false
}

func (c ConfigMap) Set(field string, val string) error {
	field = strings.ToLower(field)
	// if c[field] != "" {
	// 	return fmt.Errorf("Config field not empty: %s", field)
	// }
	c[field] = val
	return nil
}

type Config struct {
	cmpx map[string]ConfigMap
	drct map[string]string
}

func NewConfig() *Config {
	return &Config{
		cmpx: make(map[string]ConfigMap),
		drct: make(map[string]string),
	}
}

func (c Config) Get(field string) (interface{}, bool) {
	if val := c.cmpx[strings.ToLower(field)]; val != nil {
		return val, true
	}
	if val := c.drct[strings.ToLower(field)]; val != "" {
		return val, true
	}
	return nil, false
}

func (c Config) Set(field string, val interface{}) error {
	if val, _ := c.Get(field); val != nil {
		return fmt.Errorf("Config field not empty: %s", field)
	}
	if val, ok := val.(ConfigMap); ok {
		c.cmpx[strings.ToLower(field)] = val
		return nil
	}
	if val, ok := val.(string); ok {
		c.drct[strings.ToLower(field)] = val
		return nil
	}
	return fmt.Errorf("Config field not a string or ConfigMap: %s", field)
}

func (c Config) GetField(parent string, field string) (interface{}, bool) {
	field = strings.ToLower(field)
	if parentMap, ok := c.Get(parent); ok {
		if val, ok := parentMap.(ConfigMap).Get(field); ok {
			return val, true
		}
	}
	return nil, false
}

func (c Config) SetField(section string, field string, val string) error {
	section = strings.ToLower(section)
	if _, ok := c.GetField(section, field); !ok {
		c.Set(section, make(ConfigMap))
	}
	return c.cmpx[section].Set(field, val)
}

func (c *Config) Sprint() string {
	var buffer strings.Builder
	for k, v := range c.drct {
		buffer.WriteString(fmt.Sprintf("\t%s:\t%s\n", k, v))
	}
	for k, v := range c.cmpx {
		buffer.WriteString(fmt.Sprintf("\t%s:\n", k))
		for k2, v2 := range v {
			buffer.WriteString(fmt.Sprintf("\t\t%s:\t%s\n", k2, v2))
		}
	}
	return buffer.String()
}

func (c *Config) Print() {
	fmt.Printf("Config:\n%+v\n", c.Sprint())
}

// -----------------------------

func LoadConfig(prefix string) *Config {
	config := NewConfig()
	waitGroup := sync.WaitGroup{}
	for _, envVar := range os.Environ() {
		waitGroup.Add(1)
		go func(envVar string) {
			defer waitGroup.Done()
			parent, field, value := parseEnvVar(envVar, prefix)
			if parent == "" {
				return
			}
			if field == "" {
				if err := config.Set(parent, value); err != nil {
					log.Println(err)
				}
				return
			}
			if err := config.SetField(parent, field, value); err != nil {
				log.Println(err)
			}
		}(envVar)
	}
	waitGroup.Wait()
	return config
}

func parseEnvVar(envVar string, prefix string) (string, string, string) {
	if !strings.HasPrefix(envVar, prefix) {
		return "", "", ""
	}
	rawKey, value, err := twoSplit(envVar, "=")
	if err != nil {
		log.Println(err)
		return "", "", ""
	}
	key := string(rawKey[(len(prefix) + 1):])
	if strings.Contains(key, "_") {
		parent, field, err := twoSplit(key, "_")
		if err != nil {

			log.Println(err)
		}
		return parent, field, value
	}
	return key, "", value
}
