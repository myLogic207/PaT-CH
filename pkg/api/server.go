package api

import (
	"context"
	"crypto/tls"
	"crypto/x509"
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
	"github.com/myLogic207/PaT-CH/pkg/util"
)

const (
	serverAddrKey keyServerAddr = "serverAddr"
)

var (
	ErrConnectionRefused = errors.New("connection refused")
	ErrStartServer       = errors.New("could not start server")
	ErrStopServer        = errors.New("could not stop server")
	ErrInitServer        = errors.New("could not initialize server")
	ErrOpenInitFile      = errors.New("could not open init file")
	ErrReadCert          = errors.New("could not read certificate")
)

type keyServerAddr string

type Server struct {
	config     *ApiConfig
	router     *gin.Engine
	server     *http.Server
	ctx        context.Context
	sessionCtl *SessionControl
	logger     *log.Logger
	running    bool
}

func NewServer(ctx context.Context, logger *log.Logger, db UserTable, config *util.ConfigMap, rc *util.ConfigMap) (*Server, error) {
	conf, err := ParseConf(config, rc)
	if err != nil {
		logger.Println(err)
	}
	return NewServerWithConf(ctx, logger, db, conf)
}

func NewServerWithConf(ctx context.Context, logger *log.Logger, db UserTable, config *ApiConfig) (*Server, error) {
	var cache sessions.Store
	secret_store := []byte("secret")
	if config.Redis {
		logger.Println("Using Redis Cache")
		conn, err := redis.NewStoreWithDB(10, "tcp", config.RedisConf.Addr(), config.RedisConf.Password, fmt.Sprint(config.RedisConf.DB), secret_store)
		if err != nil {
			logger.Println(err)
			return nil, ErrConnectionRefused
		}
		cache = conn
	} else {
		logger.Println("No cache provided, using cookie fallback")
		cache = cookie.NewStore(secret_store)
	}
	return NewServerWithCacheConf(ctx, logger, db, config, cache)
}

func NewServerWithCache(ctx context.Context, logger *log.Logger, db UserTable, conf *util.ConfigMap, cache sessions.Store) (*Server, error) {
	config, err := ParseConf(conf, nil)
	if err != nil {
		logger.Println(err)
	}
	return NewServerWithCacheConf(ctx, logger, db, config, cache)
}

// Actual server creation
func NewServerWithCacheConf(ctx context.Context, logger *log.Logger, db UserTable, config *ApiConfig, cache sessions.Store) (*Server, error) {
	config.init()
	sessionCtl := NewSessionControl(db, logger)
	router := NewRouter(sessionCtl, logger, cache)

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
		logger:     logger,
	}, nil
}

func (s *Server) Init() error {
	file, err := os.Open(s.config.InitFile)
	if err != nil {
		s.logger.Println(err)
		return ErrOpenInitFile
	}
	defer file.Close()
	// if folder, if file, if not exist, create
	var info os.FileInfo
	if info, err = file.Stat(); err != nil {
		s.logger.Println("Init file is a directory")
		return ErrOpenInitFile
	}
	if info.IsDir() {
		files, err := os.ReadDir(s.config.InitFile)
		if err != nil {
			s.logger.Println(err)
			return ErrOpenInitFile
		}
		for _, f := range files {
			if f.IsDir() {
				continue
			}
			info, err := f.Info()
			if err != nil {
				s.logger.Println(err)
				continue
			}
			if err := s.loadInitFile(fmt.Sprint(s.config.InitFile, string(os.PathSeparator), info.Name()), s.sessionCtl); err != nil {
				s.logger.Println(err)
				return ErrInitServer
			}
		}
		return nil
	}
	return s.loadInitFile(s.config.InitFile, s.sessionCtl)
}

func (s *Server) loadInitFile(path string, sessionCtl *SessionControl) error {
	if !strings.HasSuffix(path, ".json") {
		return errors.New("not a json file: " + path)
	}
	s.logger.Println("Loading init file: " + path)
	cont, err := os.ReadFile(path)
	if err != nil {
		s.logger.Println(err)
		return errors.New("could not read file: " + path)
	}
	patches, err := ParsePatches(string(cont))
	if err != nil {
		s.logger.Println(err)
		return errors.New("could not parse file: " + path)
	}
	return patches.Apply()
}

func (s *Server) Start() error {
	s.logger.Println("Starting server")
	s.running = true
	if s.config.cert == nil {
		go func() {
			if err := s.server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
				s.logger.Fatalln(err)
			}
		}()
	} else {
		if !s.config.cert.Validate() {
			return ErrStartServer
		}
		go func() {
			if err := s.server.ListenAndServeTLS(s.config.cert.Cert, s.config.cert.Key); !errors.Is(err, http.ErrServerClosed) {
				s.logger.Fatalln(err)
			}
		}()
	}
	s.logger.Println("Serving https on " + s.Addr("/"))
	return nil
}

func (s *Server) Stop() error {
	if !s.running {
		s.logger.Println("Server is not running")
		return ErrStopServer
	}
	if err := s.server.Shutdown(s.ctx); err != nil {
		s.logger.Println(err)
		return ErrStopServer
	}
	s.running = false
	s.logger.Println("Server stopped")
	return nil
}

func (s *Server) Addr(route string) string {
	pre := "http://"
	if s.config.cert != nil {
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

type Certificate struct {
	Cert string `json:"cert"`
	Key  string `json:"key"`
}

func (c *Certificate) Validate() bool {
	cert, err := tls.LoadX509KeyPair(c.Cert, c.Key)
	if err != nil {
		log.Println(err)
		return false
	}
	if cert.Leaf == nil {
		log.Println("Parsing certificate successful")
		return true
	}

	if cert.Leaf.PublicKeyAlgorithm != x509.RSA {
		log.Println("Leaf certificate is not RSA")
		return false
	}
	return false
}
