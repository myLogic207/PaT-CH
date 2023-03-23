package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/joho/godotenv/autoload"
	"github.com/mylogic207/PaT-CH/api"
	"github.com/mylogic207/PaT-CH/storage/data"
	"github.com/mylogic207/PaT-CH/system"
)

const LOGO = `
8888888888o      .8.    8888888 8888888888 ,o888888o.    8 8888        8
88888    '88.   .888.         8 8888      8888     '88.  8 8888        8
88888     '88  :88888.        8 8888   ,8 8888       '8. 8 8888        8
88888     ,88 . '88888.       8 8888   88 8888           8 8888        8
88888.   ,88'.8. '88888.      8 8888   88 8888           8 8888        8
8888888888P'.8'8. '88888.     8 8888   88 8888           8 8888        8
88888      .8' '8. '88888.    8 8888   88 8888           8 8888888888888
88888     .8'   '8. '88888.   8 8888   '8 8888       .8' 8 8888        8
88888    .888888888. '88888.  8 8888      8888     ,88'  8 8888        8
88888   .8'       '8. '88888. 8 8888       '8888888P'    8 8888        8
`

func prepareData(config *system.Config) (*system.ConfigMap, *system.ConfigMap, error) {
	log.Println("loading data configs...")
	var dbConf, redisConf *system.ConfigMap
	if val, ok := config.Get("db"); ok {
		if cMap, ok := val.(*system.ConfigMap); ok {
			dbConf = cMap
		}
	} else {
		return nil, nil, errors.New("db config not found")
	}
	if val, ok := config.Get("redis"); ok {
		if cMap, ok := val.(*system.ConfigMap); ok {
			redisConf = cMap
		}
	} else {
		return nil, nil, errors.New("redis config not found")
	}
	log.Println("data configs loaded")
	return dbConf, redisConf, nil
}

func prepareApi(config *system.Config) (*system.ConfigMap, *system.ConfigMap, error) {
	log.Println("loading api configs...")
	var apiConf, redisConf *system.ConfigMap
	if val, ok := config.Get("api"); ok {
		if cMap, ok := val.(*system.ConfigMap); ok {
			apiConf = cMap
		} else {
			return nil, nil, errors.New("api config is not a map")
		}
	} else {
		return nil, nil, errors.New("api config not found")
	}
	if val, ok := config.Get("redis"); ok {
		if cMap, ok := val.(*system.ConfigMap); ok {
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

func setWorkDir(path string) error {
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

func prepEnv() (string, int, error) {
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
		if err := setWorkDir(val); err != nil {
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

	log.Print(LOGO)

	return prefix, timeout, nil
}

func main() {
	prefix, timeout, err := prepEnv()
	if err != nil {
		log.Fatalln("error while preparing environment: ", err)
	}

	log.Println("starting ", prefix, " server...")
	mainContext, mainStop := context.WithCancel(context.Background())

	config := system.LoadConfig(prefix)
	dbConf, redisConf, err := prepareData(config)
	if err != nil {
		log.Fatalln("error while preparing data: ", err)
	}
	apiConf, cacheConf, err := prepareApi(config)
	if err != nil {
		log.Fatalln("error while preparing api: ", err)
	}

	database, err := data.NewConnector(mainContext, dbConf, redisConf)
	if err != nil {
		log.Fatalln("error while creating database connector: ", err)
	}
	server, err := api.NewServer(mainContext, database.Users, apiConf, cacheConf)
	if err != nil {
		log.Fatalln("error while creating server: ", err)
	}

	log.Println("Components initialized, starting...")
	if err := database.Init(); err == data.ErrOpenInitFile {
		log.Println("Cannot open Init-file/file not found, skipping database initialization")
	} else if err != nil {
		log.Fatalln("error while initializing database: ", err)
	}
	server.Init()
	if err := server.Start(); err == api.ErrOpenInitFile {
		log.Println("Cannot open Init-file/file not found, skipping server initialization")
	} else if err != api.ErrStartServer {
		log.Fatalln("error while starting server: ", err)
	}
	defer server.Stop()
	log.Println("Server started")

	if timeout > 0 {
		// timeout in seconds
		time.Sleep(time.Duration(timeout) * time.Second)
		mainStop()
	}
	for {
		time.Sleep(time.Duration(1<<63 - 1))
	}
}
