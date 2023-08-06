package setup

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/myLogic207/PaT-CH/pkg/util"
)

func PrepareDatabase(config *util.Config) (*util.ConfigMap, *util.ConfigMap, error) {
	log.Println("loading database configs...")
	var dbConf, redisConf *util.ConfigMap
	if val, ok := config.Get("db"); ok {
		if cMap, ok := val.(*util.ConfigMap); ok {
			dbConf = cMap
		}
	} else {
		return nil, nil, errors.New("database config not found")
	}
	if val, ok := config.Get("redis"); ok {
		if cMap, ok := val.(*util.ConfigMap); ok {
			redisConf = cMap
		}
	} else {
		return nil, nil, errors.New("redis config not found")
	}
	log.Println("data configs loaded")
	return dbConf, redisConf, nil
}

func PrepareApi(config *util.Config) (*util.ConfigMap, *util.ConfigMap, error) {
	log.Println("loading api configs...")
	var apiConf, redisConf *util.ConfigMap
	if val, ok := config.Get("api"); ok {
		if cMap, ok := val.(*util.ConfigMap); ok {
			apiConf = cMap
		} else {
			return nil, nil, errors.New("api config is not a map")
		}
	} else {
		return nil, nil, errors.New("api config not found")
	}
	if val, ok := config.Get("redis"); ok {
		if cMap, ok := val.(*util.ConfigMap); ok {
			redisConf = cMap
		} else {
			return nil, nil, errors.New("redis config is not a map")
		}
	} else {
		return nil, nil, errors.New("redis config not found")
	}
	if val, ok := apiConf.Get("redisdb"); ok {
		redisConf.Set("db", val)
	} else {
		log.Println("redis db not set, using default")
	}
	log.Println("api configs loaded")
	return apiConf, redisConf, nil
}

func PrepareWorkDir(path string) error {
	log.Println("setting working directory to ", path)
	file, err := os.Open(path)
	if err != nil {
		log.Println("error while opening working directory:\n", err)
		if strings.Contains(err.Error(), "The system cannot find the file specified.") {
			if err := os.Mkdir(path, 0777); err != nil {
				return err
			}
			time.Sleep(1 * time.Second)
			if err := os.Chdir(path); err != nil {
				return err
			}
		} else {
			return err
		}
	}
	file.Close()
	if err := os.Chdir(path); err != nil {
		return err
	}
	return nil
}

func PrepareEnvironment() (string, int, error) {
	log.Println("loading environment...")
	if val, ok := os.LookupEnv("ENVIRONMENT"); ok {
		log.Println("starting server in ", val, " mode")
	}
	var prefix string
	if val, ok := os.LookupEnv("PREFIX"); ok {
		prefix = strings.ToUpper(val)
	} else {
		return "", -1, errors.New("prefix not set")
	}
	log.Println("environment loaded")

	if val, ok := os.LookupEnv(fmt.Sprintf("%s_DIR", prefix)); ok {
		if err := PrepareWorkDir(val); err != nil {
			return "", -1, err
		}
	} else {
		return "", -1, errors.New("working directory not set")
	}

	timeout := 0
	val, ok := os.LookupEnv(fmt.Sprintf("%s_TIMEOUT", prefix))
	if ok {
		var err error
		timeout, err = strconv.Atoi(val)
		if err != nil {
			return "", -1, err
		}
	}

	return prefix, timeout, nil
}
