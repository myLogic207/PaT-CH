package api

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"sync"

	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"
	"github.com/mylogic207/PaT-CH/system"
)

var logger = log.New(log.Writer(), "api: ", log.Flags())

type ApiConfig struct {
	Host          string `json:"host"`
	Port          uint16 `json:"port"`
	Secure        bool   `json:"secure"`
	CertFile      string `json:"cert_file"` // only used if secure is true
	KeyFile       string `json:"key_file"`  // only used if secure is true
	Redis         bool   `json:"redis"`
	RedisHost     string `json:"redis_host"`
	RedisPort     uint16 `json:"redis_port"`
	RedisPassword string `json:"redis_password"`
	RedisDB       string `json:"redis_db"`
}

func (c *ApiConfig) Addr() string {
	return c.Host + ":" + fmt.Sprint(c.Port)
}

func (c *ApiConfig) init() error {
	if c.Host == "" {
		c.Host = "localhost"
		logger.Println("could not determine host, using localhost")
	}
	if c.Port == 0 {
		c.Port = 2070
		logger.Println("could not determine port, using default")
	}
	if c.Secure {
		if c.CertFile == "" {
			return errors.New("cert file not specified")
		}
		if c.KeyFile == "" {
			return errors.New("key file not specified")
		}
	}
	if c.Redis {
		if c.RedisHost == "" {
			c.RedisHost = "localhost"
			logger.Println("could not determine redis host, using localhost")
		}
		if c.RedisPort == 0 {
			c.RedisPort = 6379
			logger.Println("could not determine redis port, using default")
		}
		if c.RedisPassword == "" {
			logger.Println("could not determine redis password, using default")
		}
		if c.RedisDB == "" {
			c.RedisDB = "0"
			logger.Println("could not determine redis db, using default")
		}
	}
	return nil
}

type Server struct {
	config     *ApiConfig
	router     *gin.Engine
	server     *http.Server
	serverWg   sync.WaitGroup
	cancel     context.CancelFunc
	ctx        context.Context
	sessionCtl SessionControl
	running    bool
}

func ParseConf(toConf *system.ConfigMap) (*ApiConfig, error) {
	config := &ApiConfig{}

	if val, ok := toConf.Get("port"); ok {
		p, _ := strconv.ParseUint(val, 10, 16)
		config.Port = uint16(p)
	}

	if val, ok := toConf.Get("host"); ok {
		config.Host = val
	}
	secure := false
	if val, ok := toConf.Get("secure"); ok {
		use, err := strconv.ParseBool(val)
		if err != nil {
			logger.Println("could not parse secure config, not using")
		} else {
			secure = use
		}
	}

	if secure {
		if val, ok := toConf.Get("cert_file"); ok {
			config.CertFile = val
		}

		if val, ok := toConf.Get("key_file"); ok {
			config.KeyFile = val
		}
	}

	if val, ok := toConf.Get("redis"); ok {
		use, err := strconv.ParseBool(val)
		if err != nil {
			logger.Println("could not parse redis config, not using")
		} else {
			config.Redis = use
		}
	}

	if val, ok := toConf.Get("redishost"); ok {
		config.RedisHost = val
	}

	if val, ok := toConf.Get("redisport"); ok {
		p, err := strconv.ParseUint(val, 10, 16)
		if err != nil {
			return config, errors.New("could not parse redis port, aborting redis connection")
		} else {
			config.RedisPort = uint16(p)
		}
	}

	if val, ok := toConf.Get("redispass"); ok {
		config.RedisPassword = val
	}

	if val, ok := toConf.Get("redisdb"); ok {
		config.RedisDB = val
	}

	logger.Println("Api finished parsing config")
	return config, nil
}

func NewServerPreConf(config *system.ConfigMap) *Server {
	conf, err := ParseConf(config)
	if err != nil {
		logger.Println(err)
	}
	return NewServer(conf)
}

type keyServerAddr string

const serverAddrKey keyServerAddr = "serverAddr"

func NewServer(config *ApiConfig) *Server {
	sessionCtl := NewSessionControl()
	config.init()
	var cache redis.Store
	if config.Redis {
		logger.Println("Using Redis Cache")
		conn, err := redis.NewStoreWithDB(10, "tcp", fmt.Sprintf("%s:%d", config.RedisHost, config.RedisPort), config.RedisPassword, config.RedisDB, []byte("secret"))
		if err != nil {
			logger.Fatal(err)
		}
		cache = conn
	} else {
		logger.Println("No cache provided, using cookie fallback")
		cache = cookie.NewStore([]byte("secret"))
	}
	router := NewRouter(sessionCtl, cache)

	ctx, cancelCtx := context.WithCancel(context.Background())
	server := &http.Server{
		Addr:    config.Addr(),
		Handler: router,
		BaseContext: func(listener net.Listener) context.Context {
			return context.WithValue(ctx, serverAddrKey, listener.Addr().String())
		},
	}

	return &Server{
		config:     config,
		router:     router,
		server:     server,
		serverWg:   sync.WaitGroup{},
		cancel:     cancelCtx,
		ctx:        ctx,
		sessionCtl: *sessionCtl,
		running:    false,
	}
}

func (s *Server) Start() {
	logger.Println("Starting server")
	s.running = true
	s.serverWg.Add(1)
	logger.Println("Serving on " + s.config.Addr())
	go func() {
		defer s.serverWg.Done()
		var service func() error
		if s.config.Secure {
			service = s.listenAndServeTLSWrapper
		} else {
			service = s.listenAndServeWrapper
		}
		if err := service(); err != http.ErrServerClosed {
			logger.Println("Server Error: ", err)
		}
		s.cancel()
	}()
}

func (s *Server) listenAndServeWrapper() error {
	return s.server.ListenAndServe()
}

func (s *Server) listenAndServeTLSWrapper() error {
	return s.server.ListenAndServeTLS(s.config.CertFile, s.config.KeyFile)
}

func (s *Server) Stop() error {
	if !s.running {
		return errors.New("server not running")
	}
	if err := s.server.Shutdown(s.ctx); err != nil {
		return err
	}
	s.serverWg.Wait()
	s.running = false
	log.Println("Server stopped")
	return nil
}

func (s *Server) Addr(route string) string {
	pre := "http://"
	if s.config.Secure {
		pre = "https://"
	}
	return pre + s.config.Addr() + route
}

func (s *Server) SetRoute(method, path string, handler gin.HandlerFunc) {
	s.router.Handle(method, path, handler)
}
