package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

type SessionLog struct {
	TimeStamp    string        `json:"time_stamp"`
	ClientIP     string        `json:"client_ip"`
	Method       string        `json:"method"`
	Path         string        `json:"path"`
	Proto        string        `json:"proto"`
	StatusCode   int           `json:"status_code"`
	Latency      time.Duration `json:"latency"`
	UserAgent    string        `json:"user_agent"`
	ErrorMessage string        `json:"error_message"`
}

var apiSkipPaths = []string{"/api/v1/health"}

func NewRouter(sessionCtl *SessionControl, cache sessions.Store) *gin.Engine {
	router := gin.New()
	cache2 := cookie.NewStore([]byte("secret"))
	router.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		Formatter: apiLogFormatter,
		Output:    logger.Writer(),
		SkipPaths: apiSkipPaths,
	}))
	cache.Options(sessions.Options{
		MaxAge:   60 * 60 * 1, // 1 hour
		SameSite: http.SameSiteLaxMode,
		Secure:   false,
	})
	if os.Getenv("ENVIRONMENT") == "development" {
		router.Use(gin.Logger())
		router.GET("/ping", routePing)
	}

	router.Use(gin.Recovery())
	router.Use(sessions.Sessions("mysession", cache2))

	addApiRoutes(router.Group("/api"), sessionCtl)

	return router
}

// /api routes
func addApiRoutes(api *gin.RouterGroup, sessionCtl *SessionControl) {
	addV1Routes(api.Group("/v1"), sessionCtl)
}

// /api/v1 routes
func addV1Routes(v1 *gin.RouterGroup, sessionCtl *SessionControl) {
	v1.GET("/health", routeHealth)
	v1.POST("/register", sessionCtl.register)
	v1.GET("/status", sessionCtl.Status)
	v1.GET("/forward/:dest", ForwardRequest)
	v1.POST("/connect", sessionCtl.Connect)
	v1.POST("/disconnect", sessionCtl.Disconnect)
	v1.GET("/session", GetID)
	addPatchRoutes(v1.Group("/patch"))
	addUserRoutes(v1.Group("/user"), sessionCtl)
}

// /api/v1/user routes
func addUserRoutes(user *gin.RouterGroup, sessionCtl *SessionControl) {
	user.Use(sessionCtl.UserRoutePass)
	user.GET("/", sessionCtl.GetUser)
	// user.POST("/", UpdateUser)
	user.DELETE("/", sessionCtl.DeleteUser)
}

// routes

func routePing(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}

// Api routes (v1)

func routeHealth(c *gin.Context) {
	// Return a simple object to indicate that the service is up.
	c.Status(http.StatusOK)
}

// GetID returns the id of the current user.
func GetID(c *gin.Context) {
	if id, ok := c.Get("id"); !ok {
		c.JSON(http.StatusForbidden, gin.H{
			"message": "disconnected",
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"id": id.(string),
		})
	}
}

// Middleware
func apiLogFormatter(params gin.LogFormatterParams) string {
	outLog := createLog(params)
	out, err := json.Marshal(outLog)
	if err != nil {
		return fmt.Sprintf("LOG ERRROR - REPORT IMMEDIATELY\n-----\n%s\n-----\n", err.Error())
	}
	return string(out) + "\n"
}

func createLog(params gin.LogFormatterParams) *SessionLog {
	outLog := &SessionLog{
		TimeStamp:  params.TimeStamp.Format("02/Jan/2006:15:04:05 -0700"),
		ClientIP:   params.ClientIP,
		Method:     params.Method,
		Path:       params.Path,
		Proto:      params.Request.Proto,
		StatusCode: params.StatusCode,
		Latency:    params.Latency,
		UserAgent:  params.Request.UserAgent(),
	}
	if params.ErrorMessage != "" {
		outLog.ErrorMessage = params.ErrorMessage
	}
	return outLog
}
