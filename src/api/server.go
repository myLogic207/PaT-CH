package api

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"
	"github.com/mylogic207/PaT-CH/system"
)

const (
	serverAddrKey keyServerAddr = "serverAddr"
)

var (
	logger               = log.New(log.Writer(), "api: ", log.Flags())
	ErrConnectionRefused = errors.New("connection refused")
	ErrStartServer       = errors.New("server starting")
	ErrStopServer        = errors.New("could not stop server")
	ErrInitServer        = errors.New("could not initialize server")
	ErrOpenInitFile      = errors.New("could not open init file")
)

type keyServerAddr string

type Server struct {
	config     *ApiConfig
	router     *gin.Engine
	server     *http.Server
	ctx        context.Context
	sessionCtl *SessionControl
	running    bool
}

func NewServer(ctx context.Context, db UserTable, config *system.ConfigMap, rc *system.ConfigMap) (*Server, error) {
	conf, err := ParseConf(config, rc)
	if err != nil {
		logger.Println(err)
	}
	return NewServerWithConf(ctx, db, conf)
}

func NewServerWithConf(ctx context.Context, db UserTable, config *ApiConfig) (*Server, error) {
	var cache redis.Store
	if config.Redis {
		logger.Println("Using Redis Cache")
		conn, err := redis.NewStoreWithDB(10, "tcp", config.RedisConf.Addr(), config.RedisConf.Password, fmt.Sprint(config.RedisConf.DB), []byte("secret"))
		if err != nil {
			logger.Println(err)
			return nil, ErrConnectionRefused
		}
		cache = conn
	} else {
		logger.Println("No cache provided, using cookie fallback")
		cache = cookie.NewStore([]byte("secret"))
	}
	return NewServerWithCacheConf(ctx, db, config, cache)
}

func NewServerWithCache(ctx context.Context, db UserTable, conf *system.ConfigMap, cache sessions.Store) (*Server, error) {
	config, err := ParseConf(conf, nil)
	if err != nil {
		logger.Println(err)
	}
	return NewServerWithCacheConf(ctx, db, config, cache)
}

// Actual server creation
func NewServerWithCacheConf(ctx context.Context, db UserTable, config *ApiConfig, cache sessions.Store) (*Server, error) {
	config.init()
	sessionCtl := NewSessionControl(db)
	router := NewRouter(sessionCtl, cache)

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
		ctx:        ctx,
		sessionCtl: sessionCtl,
		running:    false,
	}, nil
}

func (s *Server) Init() error {
	file, err := os.Open(s.config.InitFile)
	if err != nil {
		logger.Println(err)
		return ErrOpenInitFile
	}
	defer file.Close()
	// if folder, if file, if not exist, create
	var info os.FileInfo
	if info, err = file.Stat(); err != nil {
		logger.Println("Init file is a directory")
		return ErrOpenInitFile
	}
	if info.IsDir() {
		files, err := os.ReadDir(s.config.InitFile)
		if err != nil {
			logger.Println(err)
			return ErrOpenInitFile
		}
		for _, f := range files {
			if f.IsDir() {
				continue
			}
			info, err := f.Info()
			if err != nil {
				logger.Println(err)
				continue
			}
			if err := loadInitFile(fmt.Sprint(s.config.InitFile, string(os.PathSeparator), info.Name()), s.sessionCtl); err != nil {
				logger.Println(err)
				return ErrInitServer
			}
		}
		return nil
	}
	return loadInitFile(s.config.InitFile, s.sessionCtl)
}

func loadInitFile(path string, sessionCtl *SessionControl) error {
	if !strings.HasSuffix(path, ".json") {
		return errors.New("not a json file: " + path)
	}
	logger.Println("Loading init file: " + path)
	cont, err := os.ReadFile(path)
	if err != nil {
		logger.Println(err)
		return errors.New("could not read file: " + path)
	}
	patches, err := ParsePatches(string(cont))
	if err != nil {
		logger.Println(err)
		return errors.New("could not parse file: " + path)
	}
	return patches.Apply()
}

func (s *Server) Start() error {
	logger.Println("Starting server")
	s.running = true
	go func() {
		if err := s.server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			logger.Fatalln(err)
		}
	}()
	logger.Println("Serving http on " + s.Addr("/"))
	// TODO: Add https support (as secondary server)
	// if s.config.SPort == 0 || s.config.CertFile == "" || s.config.KeyFile == "" {
	// 	return ErrStartServer
	// }
	// go func() {
	// 	if err := s.server.ListenAndServeTLS(s.config.CertFile, s.config.KeyFile); !errors.Is(err, http.ErrServerClosed) {
	// 		logger.Fatalln(err)
	// 	}
	// }()
	// logger.Println("Serving https on " + fmt.Sprint(s.config.SPort))
	return ErrStartServer
}

func (s *Server) Stop() error {
	if !s.running {
		logger.Println("Server is not running")
		return ErrStopServer
	}
	if err := s.server.Shutdown(s.ctx); err != nil {
		logger.Println(err)
		return ErrStopServer
	}
	s.running = false
	log.Println("Server stopped")
	return nil
}

func (s *Server) Addr(route string) string {
	pre := "http://"
	return pre + s.config.Addr() + route
}

func (s *Server) SetRoute(method, path string, handler gin.HandlerFunc) {
	s.router.Handle(method, path, handler)
}

func (s *Server) GetContext() context.Context {
	return s.ctx
}
