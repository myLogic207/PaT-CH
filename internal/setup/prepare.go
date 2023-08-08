package setup

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

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

	return prefix, timeout, nil
}
