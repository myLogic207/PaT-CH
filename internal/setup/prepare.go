package setup

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/myLogic207/PaT-CH/pkg/util"
)

var SUB_DIRS = []string{"logs"}

func ensureDir(path string) error {
	if fileInfo, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			if err := os.Mkdir(path, 0777); err != nil {
				return err
			}
		}
	} else if !fileInfo.IsDir() || fileInfo.Mode().Perm() != 0777 {
		return errors.New("directory is not accessible")
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

func PrepareSubsystemInit(prefix string, subsystemName string, additionalSystems []string, mainConfig *util.Config) (*log.Logger, *util.Config, error) {
	logger, err := util.CreateLogger(fmt.Sprintf("%s [%s]", prefix, subsystemName))
	if err != nil {
		return nil, nil, err
	}
	config, ok := mainConfig.Get(subsystemName).(*util.Config)
	if !ok {
		return nil, nil, errors.New("error while loading " + subsystemName + " config")
	}

	for _, system := range additionalSystems {
		if subConfig, ok := mainConfig.GetConfig(system); ok {
			config.MergeInConfig(system, subConfig)
		} else {
			logger.Printf("warning: %s config not found, not using %s\n", system, system)
		}
	}

	return logger, config, nil
}
