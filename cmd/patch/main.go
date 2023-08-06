package main

import (
	"context"
	"log"
	"time"

	_ "github.com/joho/godotenv/autoload"
	"github.com/myLogic207/PaT-CH/internal/setup"
	"github.com/myLogic207/PaT-CH/pkg/api"
	"github.com/myLogic207/PaT-CH/pkg/storage/data"
	"github.com/myLogic207/PaT-CH/pkg/util"
)

func main() {
	prefix, timeout, err := setup.PrepareEnvironment()
	if err != nil {
		log.Fatalln("error while preparing environment: ", err)
	}

	log.Print(setup.LOGO)

	log.Println("starting ", prefix, " server...")
	mainContext, mainStop := context.WithCancel(context.Background())

	config := util.LoadConfig(prefix)
	dbConf, redisConf, err := setup.PrepareDatabase(config)
	if err != nil {
		log.Fatalln("error while preparing data: ", err)
	}
	apiConf, cacheConf, err := setup.PrepareApi(config)
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

	log.Println("Components setupialized, starting...")
	if err := database.Init(); err == data.ErrOpenInitFile {
		log.Println("Cannot open setup-file/file not found, skipping database setupialization")
	} else if err != nil {
		log.Fatalln("error while setupializing database: ", err)
	}
	server.Init()
	if err := server.Start(); err == api.ErrOpenInitFile {
		log.Println("Cannot open setup-file/file not found, skipping server setupialization")
	} else if err != nil {
		log.Fatalln("error while starting server:", err)
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
