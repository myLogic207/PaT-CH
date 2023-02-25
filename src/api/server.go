package api

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"

	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"
	"github.com/mylogic207/PaT-CH/system"
)

const serverAddrKey keyServerAddr = "serverAddr"

var logger = log.New(log.Writer(), "api: ", log.Flags())

type keyServerAddr string

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

func NewServer(config *system.ConfigMap, rc *system.ConfigMap) (*Server, error) {
	conf, err := ParseConf(config, rc)
	if err != nil {
		logger.Println(err)
	}
	return NewServerWithConf(conf)
}

func NewServerWithConf(config *ApiConfig) (*Server, error) {
	sessionCtl := NewSessionControl()
	config.init()
	var cache redis.Store
	if config.Redis {
		logger.Println("Using Redis Cache")
		conn, err := redis.NewStoreWithDB(10, "tcp", config.RedisConf.Addr(), config.RedisConf.Password, fmt.Sprint(config.RedisConf.DB), []byte("secret"))
		if err != nil {
			return nil, err
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
	}, nil
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

func (s *Server) GetContext() context.Context {
	return s.ctx
}
