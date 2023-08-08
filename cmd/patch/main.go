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
	log.Println("starting System...")
	prefix, timeout, err := setup.PrepareEnvironment()
	if err != nil {
		log.Fatalln("error while preparing environment: ", err)
	}
	logger, err := setup.PrepareLogger(prefix)
	if err != nil {
		log.Fatalln("error while preparing logger: ", err)
	}

	logger.Print(setup.LOGO)

	logger.Println("starting ", prefix, " server...")
	mainContext, mainStop := context.WithCancel(context.Background())

	config := util.LoadConfig(prefix)
	dbConf, redisConf, err := setup.PrepareDatabase(config)
	if err != nil {
		logger.Fatalln("error while preparing data: ", err)
	}
	apiConf, cacheConf, err := setup.PrepareApi(config)
	if err != nil {
		logger.Fatalln("error while preparing api: ", err)
	}

	database, err := data.NewConnector(mainContext, dbConf, redisConf)
	if err != nil {
		logger.Fatalln("error while creating database connector: ", err)
	}
	server, err := api.NewServer(mainContext, database.Users, apiConf, cacheConf)
	if err != nil {
		logger.Fatalln("error while creating server: ", err)
	}

	logger.Println("Components set up and initialized, starting...")
	if err := database.Init(); err == data.ErrOpenInitFile {
		logger.Println("Cannot open setup-file/file not found, skipping database initialization")
	} else if err != nil {
		logger.Fatalln("error while initializing database: ", err)
	}
	server.Init()
	if err := server.Start(); err == api.ErrOpenInitFile {
		logger.Println("Cannot open setup-file/file not found, skipping server initialization")
	} else if err != nil {
		logger.Fatalln("error while starting server:", err)
	}
	defer server.Stop()
	logger.Println("Server started")

	if timeout > 0 {
		// timeout in seconds
		time.Sleep(time.Duration(timeout) * time.Second)
		mainStop()
	}
	for {
		time.Sleep(time.Duration(1<<63 - 1))
	}
}
