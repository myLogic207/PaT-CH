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
	RedisPort     uint16 `json:"redis_port"`
	RedisPassword string `json:"redis_password"`
	RedisDB       string `json:"redis_db"`
}

func (c ApiConfig) Addr() string {
	return c.Host + ":" + fmt.Sprint(c.Port)
}

type Server struct {
	config     *ApiConfig
	router     *gin.Engine
	server     *manners.GracefulServer
	sessionCtl SessionControl
	running    bool
	secure     bool
}

func ParseConf(toConf *system.ConfigMap) (*ApiConfig, error) {
	var port uint16 = 2070
	if val, ok := toConf.Get("port"); ok {
		p, err := strconv.ParseUint(val, 10, 16)
		if err != nil {
			log.Println("could not parse port, using default")
		} else {
			port = uint16(p)
		}
	} else {
		log.Println("could not determine port, using default")
	}

	host := "localhost"
	if val, ok := toConf.Get("host"); ok {
		host = val
	} else {
		log.Println("could not determine host, using localhost")
	}

	config := &ApiConfig{
		Host:  host,
		Port:  uint16(port),
		Redis: false,
	}

	if val, ok := toConf.Get("redis"); ok {
		use, err := strconv.ParseBool(val)
		if err != nil {
			log.Println("could not parse redis config, not using")
		} else {
			config.Redis = use
		}
	} else {
		log.Println("could not determine redis config, not using")
	}

	if !config.Redis {
		return config, nil
	}

	if val, ok := toConf.Get("redishost"); ok {
		config.RedisHost = val
	} else {
		return config, errors.New("could not determine redis host, aborting redis connection")
	}

	if val, ok := toConf.Get("redisport"); ok {
		p, err := strconv.ParseUint(val, 10, 16)
		if err != nil {
			return config, errors.New("could not parse redis port, aborting redis connection")
		} else {
			config.RedisPort = uint16(p)
		}
	} else {
		return config, errors.New("could not determine redis port, aborting redis connection")
	}

	if val, ok := toConf.Get("redispass"); ok {
		config.RedisPassword = val
	} else {
		log.Println("could not determine redis password, trying with none")
	}

	if val, ok := toConf.Get("redisdb"); ok {
		config.RedisDB = val
	} else {
		log.Println("could not determine redis db, using 0")
		config.RedisDB = "0"
	}

	log.Println("Api finished parsing config")

	return config, nil
}

func NewServer(config system.ConfigMap) *Server {
	conf, err := ParseConf(&config)
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
		conn, err := redis.NewStoreWithDB(10, "tcp", fmt.Sprintf("%s:%d", config.RedisHost, config.RedisPort), config.RedisPassword, config.RedisDB, []byte("secret"))
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
		config:     config,
		router:     router,
		server:     server,
		sessionCtl: *sessionCtl,
		running:    false,
		secure:     false,
	}
}

func (s *Server) Start() {
	log.Println("Starting server")
	s.running = true
	log.Println("Servering on " + s.config.Addr())
	go s.server.ListenAndServe()
}

func (s *Server) StartSecure(certFile, keyFile string) {
	s.running = true
	s.secure = true
	go s.server.ListenAndServeTLS(certFile, keyFile)
}

func (s *Server) Stop() error {
	if !s.running {
		return errors.New("server not running")
	}
	if close := s.server.Close(); close {
		log.Println("Server closing")
		s.running = false
		return nil
	} else {
		return errors.New("server already closing")
	}
}

func (s *Server) Addr(route string) string {
	pre := "http://"
	if s.secure {
		pre = "https://"
	}
	return pre + s.config.Addr() + route
}

func (s *Server) SetRoute(method, path string, handler gin.HandlerFunc) {
	s.router.Handle(method, path, handler)
}
