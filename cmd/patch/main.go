package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/joho/godotenv/autoload"
	"github.com/myLogic207/PaT-CH/internal/setup"
	"github.com/myLogic207/PaT-CH/pkg/api"
	"github.com/myLogic207/PaT-CH/pkg/storage/data"
	"github.com/myLogic207/PaT-CH/pkg/util"
)

var SYSTEM_LIST = []string{"db", "redis", "api"}

func loadApi(ctx context.Context, prefix string, mainConfig *util.Config, dbConnection api.UserTable) (*api.Server, error) {
	logger, config, err := setup.PrepareSubsystemInit(prefix, "API", []string{"redis"}, mainConfig)
	if err != nil {
		return nil, err
	}

	server, err := api.NewServer(ctx, logger, config, dbConnection)
	if err != nil {
		return nil, err
	}

	if err := server.Init(); err != nil && err != api.ErrOpenInitFile {
		return nil, err
	}

	return server, nil
}

func loadDB(ctx context.Context, prefix string, mainConfig *util.Config) (*data.DataBase, error) {
	logger, config, err := setup.PrepareSubsystemInit(prefix, "DB", []string{"redis"}, mainConfig)
	if err != nil {
		return nil, err
	}

	database, err := data.NewConnector(ctx, logger, config)
	if err != nil {
		return nil, err
	}

	if err := database.Init(); err != nil && err != data.ErrOpenInitFile {
		return nil, err
	}

	return database, nil
}

func registerSignalHandlers() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		cleanup()
		os.Exit(1)
	}()
}

func main() {
	mainContext, mainStop := context.WithCancel(context.Background())
	defer cleanup(mainStop)
	// Load and prepare configs
	registerSignalHandlers()
	prefix, timeout, err := setup.PrepareEnvironment()
	if err != nil {
		log.Fatalln("error while preparing environment: ", err)
	}

	mainConfig, err := util.LoadConfig(prefix, nil)
	if err != nil {
		log.Fatalln("error while loading config: ", err)
	}
	log.Println("starting System...")

	if logConfig, ok := mainConfig.GetConfig("log"); ok {
		if _, ok := logConfig.GetString("prefix"); !ok {
			if err := logConfig.Set("prefix", prefix); err != nil {
				log.Println("error setting default log prefix")
			}
		}
		util.SetDefaultLogger(logConfig)
	}

	logger, err := util.CreateLogger(prefix)
	if err != nil {
		log.Fatalln("error while preparing logger: ", err)
	}

	logger.Print(setup.LOGO)
	logger.Println("starting", prefix, "server...")

	// Load and prepare components
	// Load DB
	database, err := loadDB(mainContext, prefix, mainConfig)
	if err != nil {
		logger.Fatalln("error while loading database: ", err)
	}

	// Load API Server
	server, err := loadApi(mainContext, prefix, mainConfig, database.Users)
	if err != nil {
		logger.Fatalln("error while loading api server: ", err)
	}

	if err := server.Start(); err != nil {
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

func cleanup(cancelCtx ...context.CancelFunc) {
	// End function
	log.Println("exiting...")
	if len(cancelCtx) > 0 {
		for _, stop := range cancelCtx {
			stop()
		}
	}
	// terminate loggers
	util.TerminateLoggers()
	// capture exit error
	if err := recover(); err != nil {
		log.Fatalln("error while exiting: ", err.(error).Error())
	}
}
