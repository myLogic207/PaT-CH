package system

import (
	"fmt"
	"log"
	"os"
	"strings"
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

func (c ConfigMap) Get(field string) (string, error) {
	if val := c[strings.ToLower(field)]; val != "" {
		return val, nil
	}
	return "", fmt.Errorf("Config Field not found: %s", field)
}

func (c ConfigMap) Set(field string, val string) error {
	field = strings.ToLower(field)
	// if c[field] != "" {
	// 	return fmt.Errorf("Config field not empty: %s", field)
	// }
	c[field] = val
	return nil
}

type Config map[string]ConfigMap

func (c Config) Get(field string) (ConfigMap, error) {
	if val := c[strings.ToLower(field)]; val != nil {
		return val, nil
	}
	return nil, fmt.Errorf("Config Field not found: %s", field)
}

func (c Config) Set(field string, val interface{}) error {
	if val, _ := c.Get(field); val != nil {
		return fmt.Errorf("Config field not empty: %s", field)
	}
	c[strings.ToLower(field)] = val.(ConfigMap)
	return nil
}

func (c Config) GetField(parent string, field string) (interface{}, error) {
	field = strings.ToLower(field)
	parentMap, err := c.Get(parent)
	if err != nil {
		return nil, err
	}
	if val := parentMap[field]; val != "" {
		return val, nil
	}
	return nil, fmt.Errorf("configuration not found: %s->%s", parent, field)
}

func (c Config) SetField(section string, field string, val string) error {
	section = strings.ToLower(section)
	if _, err := c.Get(section); err != nil {
		c[section] = make(ConfigMap)
	}
	return c[section].Set(field, val)
}

func (c *Config) Print() {
	var buffer strings.Builder
	for k, v := range *c {
		buffer.WriteString(fmt.Sprintf("\t%s:\n", k))
		for k2, v2 := range v {
			buffer.WriteString(fmt.Sprintf("\t\t%s:\t%s\n", k2, v2))
		}
	}
	fmt.Printf("Config:\n%+v\n", buffer.String())
}

// -----------------------------

func LoadConfig(prefix string) Config {
	config := make(Config)
	for _, envVar := range os.Environ() {
		if !strings.HasPrefix(envVar, prefix) {
			continue
		}
		rawKey, value, err := twoSplit(envVar, "=")
		if err != nil {
			log.Println(err)
			continue
		}
		key := string(rawKey[(len(prefix) + 1):])
		parent, field, err := twoSplit(key, "_")
		if err != nil {
			log.Println(err)
		}
		config.SetField(parent, field, value)
	}
	return config
}
