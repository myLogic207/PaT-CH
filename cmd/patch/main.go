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

var SYSTEM_LIST = []string{"db", "redis", "api"}

func main() {
	log.Println("starting System...")
	prefix, timeout, err := setup.PrepareEnvironment()
	if err != nil {
		log.Fatalln("error while preparing environment: ", err)
	}
	logger, err := util.CreateLogger(prefix)
	if err != nil {
		log.Fatalln("error while preparing logger: ", err)
	}

	logger.Print(setup.LOGO)

	logger.Println("starting ", prefix, " server...")
	mainContext, mainStop := context.WithCancel(context.Background())

	// Load and prepare configs
	configLogger, err := util.CreateLogger(prefix + " [CONFIG]")
	if err != nil {
		logger.Fatalln("error while preparing logger: ", err)
	}
	config := util.LoadConfigWithLogger(prefix, configLogger)
	if config == nil {
		logger.Fatalln("error while loading config")
	}
	configMaps, err := util.LoadConfigMaps(config, SYSTEM_LIST)
	if err != nil {
		logger.Fatalln("error while loading config maps: ", err)
	}

	// Load and prepare components
	// Load DB
	dbLogger, err := util.CreateLogger(prefix + " [DB]")
	if err != nil {
		logger.Fatalln("error while preparing logger: ", err)
	}
	database, err := data.NewConnector(mainContext, dbLogger, configMaps["db"], configMaps["redis"])
	if err != nil {
		logger.Fatalln("error while creating database connector: ", err)
	}

	// Load API
	apiLogger, err := util.CreateLogger(prefix + " [API]")
	if err != nil {
		logger.Fatalln("error while preparing logger: ", err)
	}
	server, err := api.NewServer(mainContext, apiLogger, database.Users, configMaps["api"], configMaps["redis"])
	if err != nil {
		logger.Fatalln("error while creating server: ", err)
	}

	// Initialize and start
	logger.Println("Configs and Components loaded, starting...")
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
	logger.Println("Server started")

	defer server.Stop()
	if timeout > 0 {
		// timeout in seconds
		time.Sleep(time.Duration(timeout) * time.Second)
		mainStop()
	}
	for {
		time.Sleep(time.Duration(1<<63 - 1))
	}
}
