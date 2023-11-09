package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/myLogic207/PaT-CH/pkg/api/internal"
)

var apiSkipPaths = []string{"/api/v1/health"}

type ApiLog struct {
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

func NewRouter(logger *log.Logger, cache sessions.Store, args ...any) *gin.Engine {
	router := gin.New()
	router.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		Formatter: generateLogFormatter,
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
	router.Use(sessions.Sessions("patch_session", cache))

	addPatchRoutes(router.Group("/patch"))
	internal.AddRoutes(router.Group("/"), args...)

	return router
}

func routePing(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}

func generateLogFormatter(params gin.LogFormatterParams) string {
	outLog := createLog(params)
	out, err := json.Marshal(outLog)
	if err != nil {
		return fmt.Sprintf("LOG ERROR - REPORT IMMEDIATELY\n-----\n%s\n-----\n", err.Error())
	}
	return string(out) + "\n"
}

func createLog(params gin.LogFormatterParams) *ApiLog {
	outLog := &ApiLog{
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
