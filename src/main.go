package main

import (
	"errors"
	"log"
	"time"

	_ "github.com/joho/godotenv/autoload"
	"github.com/mylogic207/PaT-CH/api"
	"github.com/mylogic207/PaT-CH/storage/cache"
	"github.com/mylogic207/PaT-CH/storage/data"
	"github.com/mylogic207/PaT-CH/system"
)

func prepareData(config *system.Config) (*system.ConfigMap, *system.ConfigMap, error) {
	var dbConf, redisConf system.ConfigMap
	if val, ok := config.Get("db"); ok {
		if cMap, ok := val.(system.ConfigMap); ok {
			dbConf = cMap
		}
	} else {
		return nil, nil, errors.New("db config not found")
	}
	if val, ok := config.Get("redis"); ok {
		if cMap, ok := val.(system.ConfigMap); ok {
			redisConf = cMap
		}
	} else {
		return nil, nil, errors.New("redis config not found")
	}
	return &dbConf, &redisConf, nil
}

func prepareApi(config *system.Config) (*system.ConfigMap, *system.ConfigMap, error) {
	var apiConf, redisConf system.ConfigMap
	if val, ok := config.Get("api"); ok {
		if cMap, ok := val.(system.ConfigMap); ok {
			apiConf = cMap
		} else {
			return nil, nil, errors.New("api config is not a map")
		}
	} else {
		return nil, nil, errors.New("api config not found")
	}
	if val, ok := config.Get("redis"); ok {
		if cMap, ok := val.(system.ConfigMap); ok {
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
	return &apiConf, &redisConf, nil
}

func main() {
	config := system.LoadConfig("PATCH")
	dbConf, redisConf, err := prepareData(config)
	if err != nil {
		panic(err)
	}
	apiConf, cacheConf, err := prepareApi(config)
	if err != nil {
		panic(err)
	}
	server, err := api.NewServer(apiConf, cacheConf)
	if err != nil {
		panic(err)
	}
	database, err := data.NewConnector(dbConf, server.GetContext())
	if err != nil {
		panic(err)
	}
	cache, err := cache.NewConnector(redisConf, server.GetContext())
	if err != nil {
		panic(err)
	}
	server.Start()
	defer server.Stop()
	time.Sleep(10000 * time.Second)
	// print("done")
}
