package util

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	ErrNotSplitAble = errors.New("cannot split into minimum (2) parts")
	ErrConfNotEmpty = errors.New("config field not empty")
	ErrFieldNil     = errors.New("supplied field cannot be nil")
)

func twoSplit(variable string, splitter string) (string, string, error) {
	split := strings.Split(variable, splitter)
	if len(split) < 2 {
		return "", "", ErrNotSplitAble
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
	logger *log.Logger
	sync.RWMutex
	nested map[string]*ConfigMap
	direct map[string]string
}

type ConfEntry struct {
	parent string
	field  string
	value  string
}

func NewConfig() *Config {
	return &Config{
		logger: log.New(os.Stdout, "[Config] ", log.LstdFlags),
		nested: make(map[string]*ConfigMap),
		direct: make(map[string]string),
	}
}

func NewConfigWithLogger(logger *log.Logger) *Config {
	return &Config{
		logger: logger,
		nested: make(map[string]*ConfigMap),
		direct: make(map[string]string),
	}
}

func (c *Config) Get(field string) (interface{}, bool) {
	c.RLock()
	defer c.RUnlock()
	if val, ok := c.nested[strings.ToLower(field)]; ok { // {
		return val, true
	}
	if val, ok := c.direct[strings.ToLower(field)]; ok {
		return val, true
	}
	return nil, false
}

func (c *Config) Set(field string, value interface{}) error {
	if value == nil {
		c.logger.Println("Config field cannot be nil: ", field)
		return ErrFieldNil
	}
	field = strings.ToLower(field)
	c.Lock()
	defer c.Unlock()
	switch valT := value.(type) {
	case int:
		c.direct[strings.ToLower(field)] = fmt.Sprintf("%d", valT)
	case bool:
		c.direct[strings.ToLower(field)] = fmt.Sprintf("%t", valT)
	case string:
		c.direct[strings.ToLower(field)] = valT
	case *ConfigMap:
		c.nested[strings.ToLower(field)] = valT
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
	if c.direct[section] != "" {
		c.logger.Printf("Config field already set directly: %s", section)
		return ErrConfNotEmpty
	}
	if c.nested[section] == nil {
		// c.logger.Printf("Creating new config map: %s", section)
		c.nested[section] = NewConfigMap()
	}
	if os.Getenv("ENVIRONMENT") == "development" {
		c.logger.Printf("Setting config field: %s->%s = %s", section, field, value)
	}
	return c.nested[section].Set(field, value)
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
	for k, v := range c.direct {
		buffer.WriteString(fmt.Sprintf("\t%s:\t%s\n", k, v))
	}
	for k, v := range c.nested {
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
	return LoadConfigWithLogger(prefix, log.New(os.Stdout, "[Config] ", log.LstdFlags))
}

func LoadConfigWithLogger(prefix string, logger *log.Logger) *Config {
	log.Println("Loading config from environment variables (prefix: " + prefix + ")")
	waitGroup := &sync.WaitGroup{}
	timeoutChannel := make(chan bool, 1)
	config := NewConfigWithLogger(logger)
	go func() {
		defer close(timeoutChannel)
		variableStream := getVars(prefix, waitGroup)
		entries := parseStream(variableStream, waitGroup, len(prefix))
		setEntries(entries, waitGroup, config)
		waitGroup.Wait()
	}()
	log.Println("Waiting for config to load...")
	select {
	case <-timeoutChannel:
		log.Println("Config loaded")
		return config
	case <-time.After(10 * time.Second):
		log.Fatalln("Config load timeout - partial config supplied")
		return nil
	}
}

func LoadConfigMaps(config *Config, configNames []string) (map[string]*ConfigMap, error) {
	configList := make(map[string]*ConfigMap)
	for _, configName := range configNames {
		rawConfigMap, ok := config.Get(configName)
		if !ok {
			return nil, errors.New("Config not found: " + configName)
		}
		if configMap, ok := rawConfigMap.(*ConfigMap); ok {
			configList[configName] = configMap
		} else {
			return nil, errors.New("Config not mappable: " + configName)
		}
	}
	return configList, nil
}

func parseEnvVar(envVar string, prefixLen int) *ConfEntry {
	rawKey, value, err := twoSplit(envVar, "=")
	if err != nil {
		log.Println(err)
		return nil
	}
	key := string(rawKey[(prefixLen + 1):])
	key = strings.ToUpper(key)

	if strings.HasSuffix(key, "_FILE") {
		log.Println("Loading config from file: " + value)
		if file, err := os.Open(value); err == nil {
			defer file.Close()
			value = readEnvFromFile(file)
			key = key[:len(key)-5]
		} else {
			log.Println(err)
			return nil
		}
	}

	if strings.Contains(key, "_") {
		parent, field, err := twoSplit(key, "_")
		if err != nil {
			log.Println(err)
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

func readEnvFromFile(file *os.File) string {
	var buffer strings.Builder
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		buffer.WriteString(scanner.Text())
	}
	return strings.Trim(buffer.String(), "\r\n")
}
