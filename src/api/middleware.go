package api

import (
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

func NewRouter() *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	router.GET("/ping", routePing)

	api := router.Group("/api")
	apiV1 := api.Group("/v1")

	apiV1.GET("/health", routeHealth)

	apiV1Auth := apiV1.Group("/auth")
	apiV1Auth.POST("/login", routeLogin)
	apiV1Auth.POST("/logout", routeLogout)

	return router
}

// helper functions

func getId() string {
	return "1234"
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

// Auth routes

func checkId(c *gin.Context) {
	session := sessions.Default(c)
	sessionID := session.Get("id")
	if sessionID == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "unauthorized",
		})
		c.Abort()
	}
}

func routeLogin(c *gin.Context) {
	session := sessions.Default(c)
	session.Set("id", getId())
	session.Save()
	c.JSON(http.StatusOK, gin.H{
		"message": "User Sign in successfully",
	})
}

func routeLogout(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Save()
	c.JSON(http.StatusOK, gin.H{
		"message": "User Sign out successfully",
	})
}
