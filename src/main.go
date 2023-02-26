package main

import (
	"context"
	"errors"
	"log"
	"time"

	_ "github.com/joho/godotenv/autoload"
	"github.com/mylogic207/PaT-CH/api"
	"github.com/mylogic207/PaT-CH/storage/data"
	"github.com/mylogic207/PaT-CH/system"
)

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

func main() {
	log.Println("starting PATCH...")
	mainContext, mainStop := context.WithCancel(context.Background())
	config := system.LoadConfig("PATCH")
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
	log.Println("Components initialized, starting server...")
	database.Init()
	go func() {
		if err := server.Start(); err != nil {
			log.Fatalln("error while starting server: ", err)
		}
	}()
	defer server.Stop()
	time.Sleep(10000 * time.Second)
	// print("done")
	mainStop()
}
