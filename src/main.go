package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/joho/godotenv/autoload"
	"github.com/mylogic207/PaT-CH/api"
	"github.com/mylogic207/PaT-CH/storage/data"
	"github.com/mylogic207/PaT-CH/system"
)

const LOGO = `
8 888888888o      .8.    8888888 8888888888 ,o888888o.    8 8888        8
8 8888    '88.   .888.         8 8888      8888     '88.  8 8888        8
8 8888     '88  :88888.        8 8888   ,8 8888       '8. 8 8888        8
8 8888     ,88 . '88888.       8 8888   88 8888           8 8888        8
8 8888.   ,88'.8. '88888.      8 8888   88 8888           8 8888        8
8 888888888P'.8'8. '88888.     8 8888   88 8888           8 8888        8
8 8888      .8' '8. '88888.    8 8888   88 8888           8 8888888888888
8 8888     .8'   '8. '88888.   8 8888   '8 8888       .8' 8 8888        8
8 8888    .888888888. '88888.  8 8888      8888     ,88'  8 8888        8
8 8888   .8'       '8. '88888. 8 8888       '8888888P'    8 8888        8
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

func prepEnv() (string, error) {
	log.Println("loading environment...")
	if val, ok := os.LookupEnv("ENVIRONMENT"); ok {
		log.Println("starting server in ", val, " mode")
	}
	var prefix string
	if val, ok := os.LookupEnv("PREFIX"); ok {
		prefix = strings.ToUpper(val) + "_"
	} else {
		return "", errors.New("prefix not set")
	}
	log.Println("environment loaded")

	if val, ok := os.LookupEnv(fmt.Sprintf("%sDIR", prefix)); ok {
		log.Println("setting working directory to ", val)
		if err := os.Chdir(val); err != nil {
			return "", err
		}
	}

	log.Print(LOGO)

	return prefix, nil
}

func main() {
	prefix, err := prepEnv()
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
	server, err := api.NewServer(mainContext, apiConf, cacheConf)
	if err != nil {
		log.Fatalln("error while creating server: ", err)
	}

	log.Println("Components initialized, starting...")
	if err := database.Init(); err != nil {
		log.Fatalln("error while initializing database: ", err)
	}
	if err := server.Start(); err != api.ErrStartServer {
		log.Fatalln("error while starting server: ", err)
	}
	defer server.Stop()
	log.Println("Server started")

	time.Sleep(10 * time.Second)
	// print("done")
	mainStop()
}
