package main

import (
	"os"
	"time"

	_ "github.com/joho/godotenv/autoload"
	"github.com/mylogic207/PaT-CH/api"
	"github.com/mylogic207/PaT-CH/system"
)

func main() {
	println(os.Getwd())
	config := system.LoadConfig("PATCH")
	config.Print()

	// sessionCache := cache.NewConnector(config.redisConfig)
	// err := sessionCache.Connect()
	// defer sessionCache.Close()
	// if err != nil {
	// 	log.Println(err)
	// }

	apiConfig, err := config.Get("api")
	if err != nil {
		println("Error getting API conf, using default values")
	}
	server := api.NewServer(apiConfig)
	server.Start()
	// defer server.Stop()
	time.Sleep(10000 * time.Second)
	// print("done")
}
