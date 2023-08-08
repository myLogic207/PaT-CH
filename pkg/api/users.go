package api

import (
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/myLogic207/PaT-CH/internal/system"
)

func (s *SessionControl) UserRoutePass(c *gin.Context) {
	session := sessions.Default(c)
	if session.Get("status") == "authorized" {
		c.Set("id", session.Get("id"))
		c.Set("username", session.Get("username"))
		c.Next() // continue
	} else {
		s.logger.Println("Unauthorized access to " + c.Request.URL.Path + " from " + c.ClientIP() + " with user agent " + c.Request.UserAgent())
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "unauthorized",
		})
		c.Abort()
		return
	}
}

func (s *SessionControl) GetUser(c *gin.Context) {
	var user *system.User
	if val, ok := c.Get("username"); ok {
		var err error
		if user, err = s.db.GetByName(c, val.(string)); err != nil {
			s.logger.Println(err)
			c.JSON(http.StatusNotFound, gin.H{"user": "Error finding User"})
			c.Abort()
			return
		}
	} else {
		c.JSON(http.StatusNotFound, gin.H{"user": "Error finding User"})
	}
	var id string
	if val, ok := c.Get("id"); ok {
		id = val.(string)
	} else {
		id = "id not found"
	}
	c.JSON(http.StatusOK, gin.H{
		"id":   id,
		"user": user,
	})
}

// func (s *SessionControl) UpdateUser(c *gin.Context) {
// 	user := c.Get("user").(system.User)
// 	updated, err := s.db.Update(c, &user)
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}
// 	c.JSON(http.StatusOK, gin.H{
// 		"message": "updated",
// 		"user":    updated,
// 	})
// }

func (s *SessionControl) DeleteUser(c *gin.Context) {
	var username string
	if val, ok := c.Get("username"); ok {
		username = val.(string)
	}
	s.logger.Println("deleting user: ", username)
	if err := s.db.DeleteByName(c, username); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "deleted",
	})
}

func (s *SessionControl) Status(c *gin.Context) {
	session := sessions.Default(c)
	if session.Get(id) == nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "disconnected"})
		return
	}

	s.logger.Println("user status requested: ", session.Get(id))

	c.JSON(http.StatusOK, gin.H{
		"message": "connected",
	})
}

func (s *SessionControl) GetSession(c *gin.Context) {
	session := sessions.Default(c)
	if session.Get(id) == nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "disconnected"})
		return
	}
	s.logger.Println("user session requested: ", session.Get(id))
	c.JSON(http.StatusOK, gin.H{
		"message": "connected",
		"id":      session.Get(id),
		"user":    session.Get("username"),
	})
}
