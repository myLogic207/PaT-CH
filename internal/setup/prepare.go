package setup

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
)

var ENV_LOADED = false

func CreateLogger(system_name string) (*log.Logger, error) {
	if !ENV_LOADED {
		return nil, errors.New("environment not loaded")
	}
	log.Println("preparing logger for (sub)system: ", system_name)

	writer_list := []io.Writer{}

	if env, ok := os.LookupEnv("ENVIRONMENT"); ok && strings.ToLower(env) == "development" {
		if val, ok := os.LookupEnv("DEV_NO_LOG"); !ok || strings.ToLower(val) != "true" {
			writer_list = append(writer_list, os.Stdout)
		}
	}

	file_writer, err := os.OpenFile("logs/"+system_name+".log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("error while opening log file: %w", err)
	}

	writer_list = append(writer_list, file_writer)

	writer := io.MultiWriter(writer_list...)
	logger := log.New(writer, system_name+": ", log.Ltime|log.LstdFlags|log.Lshortfile)
	log.Println("logger prepared")
	return logger, nil
}

var SUB_DIRS = []string{"logs"}

func ensureDir(path string) error {
	if fileInfo, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			if err := os.Mkdir(path, 0777); err != nil {
				return err
			}
		}
	} else if !fileInfo.IsDir() {
		return errors.New("working directory is not a directory")
	} else if fileInfo.Mode().Perm() != 0777 {
		return errors.New("working directory is not accessible")
	}
	return nil
}

func PrepareWorkDir(path string) error {
	log.Println("setting working directory to", path)
	if err := ensureDir(path); err != nil {
		return err
	}

	if err := os.Chdir(path); err != nil {
		return err
	}

	log.Println("ensuring sub directories")
	for _, subDir := range SUB_DIRS {
		if err := ensureDir(subDir); err != nil {
			return err
		}
	}

	return nil
}

func PrepareEnvironment() (string, int, error) {
	log.Println("loading environment...")
	if val, ok := os.LookupEnv("ENVIRONMENT"); ok {
		log.Println("starting server in", val, "mode")
	}
	var prefix string
	if val, ok := os.LookupEnv("PREFIX"); ok {
		prefix = strings.ToUpper(val)
	} else {
		return "", -1, errors.New("prefix not set")
	}
	log.Println("environment prefix detected")

	if val, ok := os.LookupEnv(fmt.Sprintf("%s_DIR", prefix)); ok {
		if err := PrepareWorkDir(val); err != nil {
			return "", -1, err
		}
	} else {
		return "", -1, errors.New("working directory not set")
	}

	timeout := 0
	if val, ok := os.LookupEnv(fmt.Sprintf("%s_TIMEOUT", prefix)); ok {
		var err error
		timeout, err = strconv.Atoi(val)
		if err != nil {
			return "", -1, err
		}
		log.Println("timeout set to", timeout)
	} else {
		log.Println("timeout not set, running indefinitely")
	}

	ENV_LOADED = true
	return prefix, timeout, nil
}
