package api

import (
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/myLogic207/PaT-CH/internal/system"
)

func (s *SessionControl) UserRoutePass(c *gin.Context) {
	session := sessions.Default(c)
	if session.Get("status") == "unauthorized" {
		logger.Println("Unauthorized access to " + c.Request.URL.Path + " from " + c.ClientIP() + " with user agent " + c.Request.UserAgent())
		c.JSON(http.StatusForbidden, gin.H{
			"message": "unauthorized",
		})
		c.Abort()
		return
	} else if session.Get("status") == "authorized" {
		c.Set("id", session.Get("id"))
		c.Set("username", session.Get("username"))
		c.Next() // continue
	} else {
		c.JSON(http.StatusForbidden, gin.H{
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
			logger.Println(err)
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
	if err := s.db.DeleteByName(c, username); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "deleted",
	})
}
