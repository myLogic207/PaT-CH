package util

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	logger           = log.New(os.Stdout, "system: ", log.LstdFlags)
	ErrNotSplittable = errors.New("cannot split into minimum (2) parts")
	ErrConfNotEmpy   = errors.New("config field not empty")
	ErrFieldNil      = errors.New("supplied field cannot be nil")
)

func twoSplit(variable string, splitter string) (string, string, error) {
	split := strings.Split(variable, splitter)
	if len(split) < 2 {
		return "", "", ErrNotSplittable
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

type ConfEntry struct {
	parent string
	field  string
	value  string
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
	if value == nil {
		logger.Println("Config field cannot be nil: ", field)
		return ErrFieldNil
	}
	field = strings.ToLower(field)
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
	c.Lock()
	defer c.Unlock()
	if c.drct[section] != "" {
		logger.Printf("Config field already set directly: %s", section)
		return ErrConfNotEmpy
	}
	if c.cmpx[section] == nil {
		// logger.Printf("Creating new config map: %s", section)
		c.cmpx[section] = NewConfigMap()
	}
	if os.Getenv("ENVIRONMENT") == "development" {
		logger.Printf("Setting config field: %s->%s = %s", section, field, value)
	}
	return c.cmpx[section].Set(field, value)
}

func (c *Config) SetEntry(entry *ConfEntry) error {
	if entry.field == "" {
		return c.Set(entry.parent, entry.value)
	}
	return c.SetField(entry.parent, entry.field, entry.value)
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

func getVars(prefix string, wg *sync.WaitGroup) <-chan string {
	variableStream := make(chan string, len(os.Environ()))
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(variableStream)
		wg.Add(len(os.Environ()))
		for _, envVar := range os.Environ() {
			defer wg.Done()
			if !strings.HasPrefix(envVar, prefix) {
				continue
			}
			if strings.Split(envVar, "_")[0] != prefix {
				continue
			}
			variableStream <- envVar
		}
	}()
	return variableStream
}

func parseStream(variableStream <-chan string, wg *sync.WaitGroup, prefixLen int) <-chan *ConfEntry {
	entries := make(chan *ConfEntry, 1)
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(entries)
		for envVar := range variableStream {
			entries <- parseEnvVar(envVar, prefixLen)
		}
	}()
	return entries
}

func setEntries(entryStream <-chan *ConfEntry, wg *sync.WaitGroup, config *Config) {
	for entry := range entryStream {
		wg.Add(1)
		go func(entry *ConfEntry) {
			defer wg.Done()
			config.SetEntry(entry)
		}(entry)
	}
}

func LoadConfig(prefix string) *Config {
	logger.Println("Loading config from environment variables (prefix: " + prefix + ")")
	waitGroup := &sync.WaitGroup{}
	timeoutChannel := make(chan bool, 1)
	config := NewConfig()
	go func() {
		defer close(timeoutChannel)
		variableStream := getVars(prefix, waitGroup)
		entries := parseStream(variableStream, waitGroup, len(prefix))
		setEntries(entries, waitGroup, config)
		waitGroup.Wait()
	}()
	logger.Println("Waiting for config to load...")
	select {
	case <-timeoutChannel:
		logger.Println("Config loaded")
		return config
	case <-time.After(10 * time.Second):
		logger.Fatalln("Config load timeout - partial config supplied")
		return nil
	}
}

func parseEnvVar(envVar string, prefixLen int) *ConfEntry {
	rawKey, value, err := twoSplit(envVar, "=")
	if err != nil {
		logger.Println(err)
		return nil
	}
	key := string(rawKey[(prefixLen + 1):])
	if strings.Contains(key, "_") {
		parent, field, err := twoSplit(key, "_")
		if err != nil {

			logger.Println(err)
		}
		return &ConfEntry{
			parent: string(parent),
			field:  string(field),
			value:  value,
		}
	}
	return &ConfEntry{
		parent: key,
		value:  value,
	}
}
