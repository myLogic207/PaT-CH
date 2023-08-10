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

const SUB_SEPARATOR = "."

var (
	ErrNoConfigFile   = errors.New("no config file found")
	ErrKeyNil         = errors.New("key cannot be nil")
	ErrFieldNil       = errors.New("field cannot be nil")
	ErrFieldNotNil    = errors.New("field already populated")
	ErrNotSplitAble   = errors.New("string not split able, might not contain split character")
	ErrFieldNotConfig = errors.New("config field is not a config map")
	ErrLoadConfig     = errors.New("Config load timeout - partial config supplied")
)

type configEntry struct {
	key   []string
	value interface{}
}

type Config struct {
	sync.RWMutex
	logger *log.Logger
	// nested map[string]*Config
	store map[string]interface{}
}

func NewConfig(initValues map[string]interface{}, logger *log.Logger) *Config {
	if logger == nil {
		logger = log.Default()
	}

	conf := &Config{
		logger: logger,
		// nested: make(map[string]*Config),
		store: make(map[string]interface{}),
	}

	if err := conf.MergeDefault(initValues); err != nil {
		panic(err)
	}

	return conf
}

func LoadConfig(prefix string, logger *log.Logger) (*Config, error) {
	if logger == nil {
		logger = log.Default()
	}
	logger.Println("Loading config from environment variables (prefix: " + prefix + ")")
	finishChannel := make(chan bool, 1)
	config := NewConfig(nil, logger)
	go config.loadVarsFromEnv(prefix, finishChannel)
	logger.Println("Waiting for config to load...")
	select {
	case <-finishChannel:
		logger.Println("Config loaded")
		return config, nil
	case <-time.After(10 * time.Second):
		return nil, ErrLoadConfig
	}
}

func (c *Config) GetString(keyString string) (string, bool) {
	if str, ok := c.Get(keyString).(string); ok {
		return str, true
	} else {
		return fmt.Sprintf("%v", str), true
	}
}

func (c *Config) GetConfig(keyString string) (*Config, bool) {
	if config, ok := c.Get(keyString).(*Config); ok {
		return config, true
	}
	return nil, false
}

func (c *Config) GetBool(keyString string) bool {
	if val, ok := c.Get(keyString).(bool); ok {
		return val
	}
	return false
}

func (c *Config) Get(keyString string) interface{} {
	c.logger.Printf("Getting config field: %s", strings.ToLower(keyString))
	key := strings.Split(keyString, SUB_SEPARATOR)
	if len(key) == 0 {
		c.logger.Println("Config field key cannot be nil")
		return nil
	}
	return c.getRecursive(key)
}

func (c *Config) getRecursive(key []string) interface{} {
	c.RLock()
	defer c.RUnlock()
	currentKey := strings.ToLower(key[0])
	if len(key) == 1 {
		if val, ok := c.store[currentKey]; ok {
			return val
		}
		return nil
	}

	if config, ok := c.store[currentKey].(*Config); ok {
		return config.getRecursive(key[1:])
	}

	return nil
}

func (c *Config) Set(keyString string, value interface{}) error {
	c.logger.Printf("Setting config field: %s", strings.ToLower(keyString))
	key := strings.Split(keyString, SUB_SEPARATOR)

	keyLength := len(key)
	if keyLength == 0 {
		c.logger.Println("Config field key cannot be nil")
		return ErrKeyNil
	}

	if value == nil {
		c.logger.Println("Config field value cannot be nil")
		return ErrFieldNil
	}

	return c.setRecursive(key, value)
}

func (c *Config) setRecursive(key []string, value interface{}) error {
	c.Lock()
	currentKey := strings.ToLower(key[0])
	if len(key) == 1 {
		if c.store[currentKey] != nil {
			c.logger.Printf("Overwriting config field: %s", currentKey)
		}
		c.store[currentKey] = value
		c.Unlock()
		return nil
	}

	if c.store[currentKey] == nil {
		c.logger.Printf("Creating new config map: %s", currentKey)
		c.store[currentKey] = NewConfig(nil, c.logger)
	}
	c.Unlock()
	if config, ok := c.store[currentKey].(*Config); ok {
		return config.setRecursive(key[1:], value)
	} else {
		c.logger.Printf("Config field %s is not a config map", currentKey)
		return ErrFieldNotConfig
	}
}

func (c *Config) MergeInConfig(nestKey string, defaultConfig *Config) error {
	var currentConfig *Config
	var ok bool
	rawCurrentConfig := c.Get(nestKey)
	if currentConfig, ok = rawCurrentConfig.(*Config); !ok {
		return ErrFieldNotConfig
	} else if rawCurrentConfig != nil {
		return c.Set(nestKey, defaultConfig)
	}

	for key, rawValue := range defaultConfig.store {
		if configEntry := currentConfig.Get(key); configEntry == nil {
			if err := currentConfig.Set(key, rawValue); err != nil {
				return err
			}
		} else if configEntry, ok := configEntry.(*Config); ok {
			if value, ok := rawValue.(*Config); ok {
				if err := configEntry.MergeInConfig(key, value); err != nil {
					return err
				}
			} else {
				return ErrFieldNotConfig
			}
		} else {
			continue
		}
	}
	return nil
}

func (c *Config) MergeDefault(defaultConfig map[string]interface{}) error {
	for rawKey, rawValue := range defaultConfig {
		if val := c.Get(rawKey); val != nil {
			continue
		}
		if err := c.Set(rawKey, rawValue); err != nil {
			return err
		}
	}
	return nil
}

func (c *Config) loadVarsFromEnv(prefix string, finishChannel chan bool) {
	waitGroup := &sync.WaitGroup{}
	defer close(finishChannel)

	variableStream := getVarStream(prefix, waitGroup)
	entries := parseVarStream(variableStream, waitGroup)
	c.setEntries(entries, waitGroup)
	waitGroup.Wait()
}

func (c *Config) LoadConfigs(configNames []string) (map[string]*Config, error) {
	configList := make(map[string]*Config)
	for _, configName := range configNames {
		rawConfig := c.Get(configName)
		if rawConfig == nil {
			return nil, errors.New("Config not found: " + configName)
		}
		if config, ok := rawConfig.(*Config); ok {
			configList[configName] = config
		} else {
			return nil, errors.New("Config not parsable: " + configName)
		}
	}
	return configList, nil
}

func getVarStream(prefix string, wg *sync.WaitGroup) <-chan string {
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
			variableStream <- strings.TrimPrefix(envVar, prefix+"_")
		}
	}()
	return variableStream
}

func parseVarStream(variableStream <-chan string, wg *sync.WaitGroup) <-chan *configEntry {
	entries := make(chan *configEntry, 1)
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(entries)
		for envVar := range variableStream {
			if val, err := parseEnvVar(envVar); err == nil {
				entries <- val
			} else {
				panic(err)
			}
		}
	}()
	return entries
}

func (c *Config) setEntries(entryStream <-chan *configEntry, wg *sync.WaitGroup) {
	for entry := range entryStream {
		wg.Add(1)
		go func(entry *configEntry) {
			defer wg.Done()
			if err := c.Set(strings.Join(entry.key, SUB_SEPARATOR), entry.value); err != nil {
				panic(err)
			}
		}(entry)
	}
}

func (c *Config) Sprint() string {
	c.RLock()
	defer c.RUnlock()
	var buffer strings.Builder
	for k, v := range c.store {
		switch entry := v.(type) {
		case *Config:
			buffer.WriteString(fmt.Sprintf("%s:\n\t%s", k, entry.Sprint()))
		default:
			buffer.WriteString(fmt.Sprintf("%s:\t%s", k, entry))
			buffer.WriteString("\n")
		}
	}
	return buffer.String()
}

func (c *Config) Print() {
	fmt.Printf("Config:\n%+v\n", c.Sprint())
}

func parseEnvVar(envVar string) (*configEntry, error) {
	key, value, err := SplitOffFirst(envVar, "=")
	if err != nil {
		return nil, err
	}

	if strings.HasSuffix(key, "_FILE") {
		log.Println("Loading config from file: " + value)
		if file, err := os.Open(value); err == nil {
			defer file.Close()
			value = readEnvFromFile(file)
			key = strings.TrimSuffix(key, "_FILE")
		} else {
			return nil, err
		}
	}

	return &configEntry{
		key:   strings.Split(key, "_"),
		value: value,
	}, nil
}

func readEnvFromFile(file *os.File) string {
	var buffer strings.Builder
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		buffer.WriteString(scanner.Text())
	}
	return strings.Trim(buffer.String(), "\r\n")
}

func SplitOffFirst(variable string, splitter string) (string, string, error) {
	split := strings.Split(variable, splitter)
	if len(split) < 2 {
		return "", "", ErrNotSplitAble
	}
	firstPart, otherParts := split[0], split[1:]
	return firstPart, strings.Join(otherParts, splitter), nil
}
