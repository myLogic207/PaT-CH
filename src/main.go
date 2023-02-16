package main

import (
	"time"

	_ "github.com/joho/godotenv/autoload"
	"github.com/mylogic207/PaT-CH/api"
	"github.com/mylogic207/PaT-CH/system"
)

func main() {
	config := system.LoadConfig("PATCH")
	config.Print()

	// sessionCache := cache.NewConnector(config.redisConfig)
	// err := sessionCache.Connect()
	// defer sessionCache.Close()
	// if err != nil {
	// 	log.Println(err)
	// }

	var apiConfig *api.ApiConfig
	if val, ok := config.Get("api"); ok {
		if cMap, ok := val.(system.ConfigMap); ok {
			conf, err := api.ParseConf(&cMap)
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
	server := api.NewServerPreConf(apiConfig)
	server.Start()
	// defer server.Stop()
	time.Sleep(10000 * time.Second)
	// print("done")
}
