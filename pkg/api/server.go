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
	"path/filepath"
	"strconv"
	"strings"

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
	ErrMergeConfig       = errors.New("could not merge config")
	ErrRedisConf         = errors.New("could not parse redis config")
	ErrStartServer       = errors.New("could not start server")
	ErrStopServer        = errors.New("could not stop server")
	ErrInitServer        = errors.New("could not initialize server")
	ErrOpenInitFile      = errors.New("could not open init file")
	ErrReadCert          = errors.New("could not read certificate")
	ErrLoadInitFile      = errors.New("could not load init file")
)

type keyServerAddr string

type Server struct {
	config  *util.Config
	cert    *Certificate
	router  *gin.Engine
	server  *http.Server
	ctx     context.Context
	logger  *log.Logger
	running bool
}

var defaultConfig = map[string]interface{}{
	"host":      "127.0.0.1",
	"port":      80,
	"initFile":  "api.init.d",
	"redis.use": false,
}

func NewServer(ctx context.Context, logger *log.Logger, config *util.Config, args ...any) (*Server, error) {
	if logger == nil {
		logger = log.Default()
	}

	if config == nil {
		config = util.NewConfig(defaultConfig, logger)
	} else {
		if err := config.MergeDefault(defaultConfig); err != nil {
			logger.Println(err)
			return nil, ErrMergeConfig
		}
	}

	cache := cookie.NewStore([]byte("secret"))
	if redisConfig, ok := config.Get("redis").(*util.Config); ok {
		if redisConfig.GetBool("use") {
			logger.Println("Using redis cache")
			var err error
			if cache, err = connectRedisCache(redisConfig); err != nil {
				return nil, ErrInitServer
			}
		}
	} else {
		return nil, ErrInitServer
	}

	serverAddress := loadAddress(config)
	if serverAddress == "" {
		return nil, ErrInitServer
	}
	router := NewRouter(logger, cache, args...)
	httpServer := &http.Server{
		Addr:    serverAddress,
		Handler: router,
		BaseContext: func(listener net.Listener) context.Context {
			return context.WithValue(ctx, serverAddrKey, listener.Addr().String())
		},
	}

	return &Server{
		config:  config,
		router:  router,
		server:  httpServer,
		ctx:     ctx,
		running: false,
		logger:  logger,
		cert:    loadCert(config),
	}, nil
}

func loadAddress(config *util.Config) string {
	serverAddress, ok := config.GetString("host")
	if !ok {
		return ""
	}
	var port int
	if serverPort, ok := config.GetString("port"); ok {
		if serverPort, err := strconv.Atoi(serverPort); err == nil {
			port = serverPort
		} else {
			return ""
		}
	} else {
		return ""
	}
	return fmt.Sprintf("%s:%d", serverAddress, port)
}

func loadCert(config *util.Config) *Certificate {
	cert := &Certificate{}
	if certPath, ok := config.GetString("cert"); ok {
		if keyPath, ok := config.GetString("key"); ok {
			cert.Cert = certPath
			cert.Key = keyPath
			return cert
		}
	}
	return nil
}

func connectRedisCache(redisConfig *util.Config) (redis.Store, error) {
	redisHost, ok := redisConfig.GetString("host")
	if !ok {
		return nil, ErrRedisConf
	}
	var port uint16
	if redisPort, ok := redisConfig.Get("port").(uint16); ok {
		port = redisPort
	} else {
		return nil, ErrRedisConf
	}
	redisAddr := fmt.Sprintf("%s:%d", redisHost, port)
	redisPassword, ok := redisConfig.GetString("password")
	if !ok {
		return nil, ErrRedisConf
	}
	redisDB, ok := redisConfig.GetString("db")
	if !ok {
		return nil, ErrRedisConf
	}
	conn, err := redis.NewStoreWithDB(10, "tcp", redisAddr, redisPassword, redisDB, []byte("secret"))
	if err != nil {
		return nil, ErrConnectionRefused
	}
	return conn, nil
}

func (s *Server) Init() error {
	initFile, ok := s.config.GetString("InitFile")
	if !ok {
		s.logger.Println("No init file provided")
		return nil
	}

	if fileStat, err := os.Stat(initFile); err != nil {
		s.logger.Println("Init file does not exist")
		return nil
	} else {
		if fileStat.IsDir() {
			return s.loadInitDir(initFile)
		}
		return s.loadInitFile(initFile)
	}
}

func (s *Server) loadInitDir(folderPath string) error {
	files, err := os.ReadDir(folderPath)
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
		if err := s.loadInitFile(filepath.Join(folderPath, info.Name())); err != nil {
			s.logger.Println(err)
			return ErrLoadInitFile
		}
	}
	return nil
}

func (s *Server) loadInitFile(path string) error {
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
	if s.cert == nil {
		go func() {
			if err := s.server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
				s.logger.Fatalln(err)
			}
		}()
	} else {
		if !s.cert.Validate() {
			return ErrStartServer
		}
		go func() {
			if err := s.server.ListenAndServeTLS(s.cert.Cert, s.cert.Key); !errors.Is(err, http.ErrServerClosed) {
				s.logger.Fatalln(err)
			}
		}()
	}
	s.logger.Println("Serving on " + s.Addr("/"))
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
	if s.cert != nil {
		pre = "https://"
	}
	host, _ := s.config.GetString("host")
	port, _ := s.config.GetString("port")
	route = strings.TrimPrefix(route, "/")
	return fmt.Sprintf("%s%s:%s/%s", pre, host, port, route)
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
