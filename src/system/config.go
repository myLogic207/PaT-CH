package system

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"
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

type ConfigMap struct {
	sync.RWMutex
	conf map[string]string
}

func NewConfigMap() *ConfigMap {
	return &ConfigMap{
		conf: make(map[string]string),
	}
}

func (c *ConfigMap) Get(field string) (string, bool) {
	c.RLock()
	defer c.RUnlock()
	if val := c.conf[strings.ToLower(field)]; val != "" {
		return val, true
	}
	return "", false
}

func (c *ConfigMap) Set(field string, val string) error {
	c.Lock()
	defer c.Unlock()
	field = strings.ToLower(field)
	// if c[field] != "" {
	// 	return fmt.Errorf("Config field not empty: %s", field)
	// }
	c.conf[field] = val
	return nil
}

type Config struct {
	sync.RWMutex
	cmpx map[string]*ConfigMap
	drct map[string]string
}

func NewConfig() *Config {
	return &Config{
		cmpx: make(map[string]*ConfigMap),
		drct: make(map[string]string),
	}
}

func (c *Config) Get(field string) (interface{}, bool) {
	c.RLock()
	defer c.RUnlock()
	if val, ok := c.cmpx[strings.ToLower(field)]; ok { // {
		return val, true
	}
	if val, ok := c.drct[strings.ToLower(field)]; ok {
		return val, true
	}
	return nil, false
}

func (c *Config) Set(field string, value interface{}) error {
	if _, ok := c.cmpx[strings.ToLower(field)]; ok || c.drct[strings.ToLower(field)] != "" {
		return fmt.Errorf("Config field not empty: %s", field)
	}
	if value == nil {
		return fmt.Errorf("Config field cannot be nil: %s", field)
	}
	c.Lock()
	defer c.Unlock()
	switch valT := value.(type) {
	case int:
		c.drct[strings.ToLower(field)] = fmt.Sprintf("%d", valT)
	case bool:
		c.drct[strings.ToLower(field)] = fmt.Sprintf("%t", valT)
	case string:
		c.drct[strings.ToLower(field)] = valT
	case *ConfigMap:
		c.cmpx[strings.ToLower(field)] = valT
	default:
		return fmt.Errorf("Config field type not supported: %s", field)
	}
	return nil
}

func (c *Config) GetField(parent string, field string) (string, bool) {
	field = strings.ToLower(field)
	if parentMap, ok := c.Get(parent); ok {
		if parentMap, ok := parentMap.(*ConfigMap); ok {
			if val, ok := parentMap.Get(field); ok {
				return val, true
			}
		}
	}
	return "", false
}

func (c *Config) SetField(section string, field string, value string) error {
	section = strings.ToLower(section)
	if val, ok := c.Get(section); !ok {
		log.Printf("Creating new config section: %s", section)
		c.Set(section, NewConfigMap())
	} else {
		if _, ok := val.(string); ok {
			return fmt.Errorf("Config field in use: %s", section)
		}
	}
	log.Printf("Setting config field: %s->%s = %s", section, field, value)
	return c.cmpx[section].Set(field, value)
}

func (c *Config) Sprint() string {
	c.RLock()
	defer c.RUnlock()
	var buffer strings.Builder
	for k, v := range c.drct {
		buffer.WriteString(fmt.Sprintf("\t%s:\t%s\n", k, v))
	}
	for k, v := range c.cmpx {
		buffer.WriteString(fmt.Sprintf("\t%s:\n", k))
		for k2, v2 := range v.conf {
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
	log.Println("Loading config from environment variables (prefix: " + prefix + ")")
	config := NewConfig()
	waitGroup := sync.WaitGroup{}
	for _, envVar := range os.Environ() {
		if !strings.HasPrefix(envVar, prefix) {
			continue
		}
		waitGroup.Add(1)
		go func(envVar string) {
			defer waitGroup.Done()
			parent, field, value := parseEnvVar(envVar, len(prefix))
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
	timeoutChannel := make(chan bool, 1)
	go func() {
		defer close(timeoutChannel)
		waitGroup.Wait()
	}()
	select {
	case <-timeoutChannel:
		log.Println("Config loaded")
		return config
	case <-time.After(10 * time.Second):
		log.Println("Config load timeout - partial config supplied")
		return config
	}
}

func parseEnvVar(envVar string, prefixLen int) (string, string, string) {
	rawKey, value, err := twoSplit(envVar, "=")
	if err != nil {
		log.Println(err)
		return "", "", ""
	}
	key := string(rawKey[(prefixLen + 1):])
	if strings.Contains(key, "_") {
		parent, field, err := twoSplit(key, "_")
		if err != nil {

			log.Println(err)
		}
		return parent, field, value
	}
	return key, "", value
}
