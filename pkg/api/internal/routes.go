package internal

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/myLogic207/PaT-CH/internal/system"
	"github.com/myLogic207/PaT-CH/pkg/util"
)

const LOGIN_URL_PATH = "/api/v1/auth/connect"

// / routes
func AddRoutes(router *gin.RouterGroup, args ...any) {
	if len(args) == 0 || args[0] == nil {
		log.Fatalln("no args passed to AddRoutes")
	}

	sessionLogger, err := util.CreateLogger("sessions")
	if err != nil {
		log.Println("failed to create session logger, falling back")
		sessionLogger = log.Default()
	}

	var sessionCtl *SessionControl
	if userDB, ok := args[0].(system.UserTable); ok {
		sessionCtl = NewSessionControl(userDB, sessionLogger)
	} else {
		log.Fatalln("first arg passed to AddRoutes is not a UserTable")
	}
	addApiRoutes(router.Group("/api"), sessionCtl)
}

// /api routes
func addApiRoutes(api *gin.RouterGroup, sessionCtl *SessionControl) {
	addV1Routes(api.Group("/v1"), sessionCtl)
}

// /api/v1 routes
func addV1Routes(v1 *gin.RouterGroup, sessionCtl *SessionControl) {
	v1.GET("/health", routeHealth)
	v1.POST("/register", sessionCtl.register)
	// v1.GET("/forward/:dest", ForwardRequest)
	v1.GET("/status", sessionCtl.Status)
	addAuthRoutes(v1.Group("/auth"), sessionCtl)
}

// /api/v1/auth routes
func addAuthRoutes(auth *gin.RouterGroup, sessionCtl *SessionControl) {
	auth.Use(sessionCtl.UserRoutePass)
	auth.POST("/connect", sessionCtl.Connect)
	auth.POST("/disconnect", sessionCtl.Disconnect)
	auth.GET("/session", sessionCtl.GetSession)

	// /api/v1/auth/user routes
	auth.GET("/user", sessionCtl.GetUser)
	// user.POST("/", UpdateUser)
	auth.DELETE("/user", sessionCtl.DeleteUser)
}

// func addUserRoutes(user *gin.RouterGroup, sessionCtl *SessionControl) {
// 	user.Use(sessionCtl.UserRoutePass)
// }

// Api routes (v1)
func routeHealth(c *gin.Context) {
	// Return a simple object to indicate that the service is up.
	c.Status(http.StatusOK)
}
