package main

import (
	"time"

	_ "github.com/joho/godotenv/autoload"
	"github.com/mylogic207/PaT-CH/api"
	"github.com/mylogic207/PaT-CH/cache"
	"github.com/mylogic207/PaT-CH/system"
)

func main() {
	config := system.LoadConfig("PATCH")
	config.Print()

	var apiConfig *api.ApiConfig
	if val, ok := config.Get("api"); ok {
		if cMap, ok := val.(*system.ConfigMap); ok {
			conf, err := api.ParseConf(cMap)
			if err != nil {
				println("Error getting API conf, using default values")
				apiConfig = &api.ApiConfig{
					Host:  "localhost",
					Port:  2070,
					Redis: false,
				}
			} else {
				apiConfig = conf
			}
		}
	} else {
		println("Error getting API conf, using default values")
	}

	var redisConf *cache.RedisConfig
	if val, ok := config.Get("redis"); ok {
		if cMap, ok := val.(*system.ConfigMap); ok {
			conf, err := cache.ParseConf(cMap)
			if err != nil {
				println("Error getting redis conf, using default values")
				redisConf = &cache.RedisConfig{
					Host:       "localhost",
					Port:       6379,
					Password:   "",
					DB:         1,
					Idle:       10,
					MaxActive:  100,
					TimeoutSec: 60,
				}
			} else {
				redisConf = conf
			}
		}
	} else {
		println("Error getting redis conf, using default values")
	}

	server := api.NewServer(apiConfig)
	cache := cache.NewConnector(redisConf)

	server.Start()
	defer server.Stop()

	cache.Connect()
	defer cache.Close()
	time.Sleep(10000 * time.Second)
	// print("done")
}
