package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"reflect"
	"strings"
	"sync"

	"github.com/mylogic207/PaT-CH/api"
)

type ConfigField struct {
	key   string
	value interface{}
}

type SystemConfig struct {
}

type Config struct {
	system SystemConfig
	api    api.ApiConfig
}

func (c *Config) setField(key string, val interface{}) error {
	confVal := reflect.ValueOf(c).Elem()
	confField := confVal.FieldByName(key)

	if !confField.IsValid() {
		return fmt.Errorf("No such field: %s in obj", key)
	}

	if !confField.CanSet() {
		return fmt.Errorf("Cannot set %s field value", key)
	}

	confType := confField.Type()
	value := reflect.ValueOf(val)
	if confType != value.Type() {
		return errors.New("Provided value type didn't match obj field type")
	}
	confField.Set(value)
	return nil
}

func (c *Config) setConfigField(parent string, key string, val interface{}) error {

}

func twoSplit(variable string, splitter string) (string, string, error) {
	split := strings.Split(variable, splitter)
	if len(split) < 2 {
		return "", "", fmt.Errorf("environmental variable not declared correctly:\n%s", variable)
	}
	key, val := split[0], split[1:]
	return key, strings.Join(val, ""), nil
}

func loadConfig(prefix string) Config {
	prefix = strings.ToUpper(prefix)
	var waitGroup sync.WaitGroup
	configParts := make(map[string][]ConfigField)

	for _, envVar := range os.Environ() {
		// go func(variable string, prefix string) {
		if !strings.HasPrefix(envVar, prefix) {
			// return
			continue
		}
		// waitGroup.Add(1)
		// defer waitGroup.Done()
		rawKey, val, err := twoSplit(envVar, "=")
		if err != nil {
			log.Println(err)
			// return
			continue
		}
		fmt.Printf("k: %s \t\t v: %s\n", rawKey[:len(prefix)-1], val)
		parent, field, err := twoSplit(rawKey[:len(prefix)], "_")
		if err != nil {
			log.Println(err)
		}
		configParts[parent] = append(configParts[parent], ConfigField{key: field, value: val})
		// }(envVar, prefix)
	}

	config := &Config{}

	for key, val := range configParts {
		fmt.Println(key, val)
		for k, v := range val {
			config.setField(k, v)
		}
	}

	waitGroup.Wait()

	return config
}
