package main

import (
	"log"
	"time"

	_ "github.com/joho/godotenv/autoload"
	"github.com/mylogic207/PaT-CH/api"
	"github.com/mylogic207/PaT-CH/storage/cache"
	"github.com/mylogic207/PaT-CH/system"
)

func main() {
	config := system.LoadConfig("PATCH")
	config.Print()

	apiConfig := api.DefaultConfig()
	log.Println("Getting API conf - using default in case of error")
	if val, ok := config.Get("api"); ok {
		if cMap, ok := val.(*system.ConfigMap); ok {
			var err error
			apiConfig, err = api.ParseConf(cMap)
			if err != nil {
				log.Println("Error getting API conf, using default values")
			}
		}
	}

	redisConf := cache.DefaultConfig()
	log.Println("Getting redis conf - using default in case of error")
	if val, ok := config.Get("redis"); ok {
		if cMap, ok := val.(*system.ConfigMap); ok {
			var err error
			redisConf, err = cache.ParseConf(cMap)
			if err != nil {
				println("Error getting redis conf, using default values")
			}
		}
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
