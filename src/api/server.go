package api

import (
	"errors"
	"fmt"
	"log"
	"net/http"

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
	RedisDB       int    `json:"redis_db"`
}

func (c ApiConfig) Addr() string {
	return c.Host + ":" + fmt.Sprint(c.Port)
}

type Server struct {
	router  *gin.Engine
	server  *manners.GracefulServer
	running bool
}

func parseConf(toConf *system.ConfigMap) (*ApiConfig, error) {
	var port uint16 = 2070
	configPort, err := toConf.Get("port")
	if err != nil {
		log.Println("Could not determine port, using default")
	} else if val, ok := configPort.(uint16); ok {
		port = val
	} else {
		log.Println("Could not determine port, using default")
	}

	host := "localhost"
	cHost, err := toConf.Get("host")
	if err != nil {
		log.Println("Could not determine host, using localhost")
	} else if val, ok := cHost.(string); ok {
		host = val
	} else {
		log.Println("Could not determine host, using localhost")
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
	} else if val, ok := configRedis.(bool); ok {
		useRedis = val
	} else {
		log.Println("could not determine redis config, not using")
	}

	if !useRedis {
		return config, nil
	}

	var redisHost string
	confRedisHost, err := toConf.Get("redis_host")
	if err != nil {
		log.Println("Could not determine redis host, aborting redis connection")
		return config, errors.New("could not determine redis host, aborting redis connection")
	} else if val, ok := confRedisHost.(string); ok {
		redisHost = val
	} else {
		log.Println("Could not determine redis host, aborting redis connection")
		return config, errors.New("could not determine redis host, aborting redis connection")
	}

	redisPassword := ""
	configPassword, err := toConf.Get("redis_password")
	if err != nil {
		log.Println("Could not determine redis password, trying with none")
	} else if val, ok := configPassword.(string); ok {
		redisPassword = val
	} else {
		log.Println("Could not determine redis password, trying with none")
	}

	redisDB := 0
	configRedisDB, err := toConf.Get("redisdb")
	if err != nil {
		log.Println("Could not determine redis db, using 0")
	} else if val, ok := configRedisDB.(int); ok {
		redisDB = val
	} else {
		log.Println("Could not determine redis db, using 0")
	}

	return &ApiConfig{
		Host:          host,
		Port:          uint16(port),
		Redis:         useRedis,
		RedisHost:     redisHost,
		RedisPassword: redisPassword,
		RedisDB:       int(redisDB),
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
	router := NewRouter()
	var cache sessions.Store
	if config.Redis {
		log.Println("Using Redis Cache")
		conn, err := redis.NewStoreWithDB(10, "tcp", config.RedisHost, config.RedisPassword, fmt.Sprintf("%d", config.RedisDB))
		if err != nil {
			log.Fatal(err)
		}
		cache = conn
	} else {
		log.Println("No cache provided, using cookie fallback")
		cache = cookie.NewStore([]byte("secret"))
	}
	router.Use(sessions.Sessions("sessions", cache))

	server := manners.NewWithServer(&http.Server{
		Addr:    config.Addr(),
		Handler: router,
	})

	return &Server{
		router:  router,
		server:  server,
		running: false,
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
