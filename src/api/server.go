package api

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/braintree/manners"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"
	"github.com/mylogic207/PaT-CH/system"
)

type ApiConfig struct {
	Host          string `json:"host"`
	Port          uint16 `json:"port"`
	Redis         bool   `json:"redis"`
	RedisHost     string `json:"redis_host"`
	RedisPassword string `json:"redis_password"`
	RedisDB       string `json:"redis_db"`
}

func (c ApiConfig) Addr() string {
	return c.Host + ":" + fmt.Sprint(c.Port)
}

type Server struct {
	router     *gin.Engine
	server     *manners.GracefulServer
	sessionCtl SessionControl
	running    bool
}

func parseConf(toConf *system.ConfigMap) (*ApiConfig, error) {
	var port uint16 = 2070
	configPort, err := toConf.Get("port")
	if err != nil {
		log.Println("Could not determine port, using default")
	} else {
		p, err := strconv.ParseUint(configPort, 10, 16)
		if err != nil {
			log.Println("Could not parse port, using default")
		} else {
			port = uint16(p)
		}
	}

	host := "localhost"
	cHost, err := toConf.Get("host")
	if err != nil {
		log.Println("Could not determine host, using localhost")
	} else {
		host = cHost
	}

	config := &ApiConfig{
		Host:  host,
		Port:  uint16(port),
		Redis: false,
	}

	useRedis := false
	configRedis, err := toConf.Get("redis")
	if err != nil {
		log.Println("could not determine redis config, not using")
	} else {
		use, err := strconv.ParseBool(configRedis)
		if err != nil {
			log.Println("could not parse redis config, not using")
		} else {
			useRedis = use
		}
	}

	if !useRedis {
		return config, nil
	}

	redisHost, err := toConf.Get("redishost")
	if err != nil {
		return config, errors.New("could not determine redis host, aborting redis connection")
	}

	redisPassword, err := toConf.Get("redispass")
	if err != nil {
		log.Println("Could not determine redis password, trying with none")
	}

	redisDB, err := toConf.Get("redisdb")
	if err != nil {
		log.Println("Could not determine redis db, using 0")
		redisDB = "0"
	}

	log.Println("Api finished parsing config")

	return &ApiConfig{
		Host:          host,
		Port:          uint16(port),
		Redis:         useRedis,
		RedisHost:     redisHost,
		RedisPassword: redisPassword,
		RedisDB:       redisDB,
	}, nil
}

func NewServer(config system.ConfigMap) *Server {
	conf, err := parseConf(&config)
	if err != nil {
		log.Println(err)
	}
	return NewServerPreConf(conf)
}

func NewServerPreConf(config *ApiConfig) *Server {
	sessionCtl := NewSessionControl()
	var cache sessions.Store
	if config.Redis {
		log.Println("Using Redis Cache")
		conn, err := redis.NewStoreWithDB(10, "tcp", config.RedisHost, config.RedisPassword, config.RedisDB)
		if err != nil {
			log.Fatal(err)
		}
		cache = conn
	} else {
		log.Println("No cache provided, using cookie fallback")
		cache = cookie.NewStore([]byte("secret"))
	}
	router := NewRouter(sessionCtl, cache)

	server := manners.NewWithServer(&http.Server{
		Addr:    config.Addr(),
		Handler: router,
	})

	return &Server{
		router:     router,
		server:     server,
		sessionCtl: *sessionCtl,
		running:    false,
	}
}

func (s *Server) Start() {
	log.Println("Starting server")
	s.running = true
	go s.server.ListenAndServe()
}

func (s *Server) StartSecure(certFile, keyFile string) {
	s.running = true
	go s.server.ListenAndServeTLS(certFile, keyFile)
}

func (s *Server) Stop() error {
	if !s.running {
		return nil
	}
	if close := s.server.Close(); close {
		log.Println("Server closing")
		s.running = false
		return nil
	} else {
		return errors.New("server already closing")
	}
}

func (s *Server) SetRoute(method, path string, handler gin.HandlerFunc) {
	s.router.Handle(method, path, handler)
}
