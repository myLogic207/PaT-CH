package api

import (
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

func NewRouter(sessionCtl *SessionControl, cache sessions.Store) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(sessions.Sessions("mysession", cache))

	router.GET("/ping", routePing)

	api := router.Group("/api")
	apiV1 := api.Group("/v1")

	apiV1.GET("/health", routeHealth)
	apiV1.POST("/connect", sessionCtl.Connect)
	apiV1.POST("/disconnect", sessionCtl.Disonnect)

	apiV1Auth := apiV1.Group("/auth")
	apiV1Auth.Use(RoutePass())
	apiV1Auth.GET("/session", GetID)

	return router
}

// routes

func routePing(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}

// Api routes (v1)

func routeHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

func RoutePass() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		sessionID := session.Get("id")
		if sessionID == nil {
			c.JSON(http.StatusForbidden, gin.H{
				"message": "unauthorized",
			})
			c.Abort()
		}
	}
}

func GetID(c *gin.Context) {
	session := sessions.Default(c)
	c.JSON(http.StatusOK, gin.H{
		// This is "complicated" to test session exists (so we need to check beforehand as well?)
		"message": session.Get("id").(string),
	})
}
