package internal

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/myLogic207/PaT-CH/internal/system"
)

const (
	id_key           = "unique_user_identifier"
	id_bytes         = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"
	auth_key         = "authorization_status"
	auth_pass_string = "user_is_authorized"
)

var (
	ErrRegister        = fmt.Errorf("error registering user")
	ErrRegisterAlready = fmt.Errorf("user already exists")
	ErrConnect         = fmt.Errorf("error connecting user")
)

type SessionControl struct {
	key_len  int
	sessions map[string]sessions.Session
	db       system.UserTable
	logger   *log.Logger
}

func NewSessionControl(db system.UserTable, logger *log.Logger) *SessionControl {
	return &SessionControl{
		key_len:  16,
		db:       db,
		sessions: make(map[string]sessions.Session),
		logger:   logger,
	}
}

func (s *SessionControl) idExists(id string) bool {
	for key := range s.sessions {
		if key == id {
			return true
		}
	}
	return false
}

// todo: make this more secure
func (s *SessionControl) getNewId() string {
	var id string
	for ok := true; ok; ok = s.idExists(id) {
		// generate a new id
		// as found here https://stackoverflow.com/questions/22892120/how-to-generate-a-random-string-of-a-fixed-length-in-go/22892986#22892986
		b := make([]byte, s.key_len)
		for i := range b {
			b[i] = id_bytes[rand.Int63()%int64(len(id_bytes))]
		}
		id = string(b)
	}
	return id
}

// func (s *SessionControl) SetDB(db system.UserTable) {
// 	s.db = db
// }

func (s *SessionControl) register(c *gin.Context) {
	var raw system.RawUser
	if err := c.BindJSON(&raw); err != nil {
		s.logger.Println(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrRegister})
		return
	}
	user, err := s.db.Create(c, raw.Username, "", raw.Password)
	if err != nil {
		s.logger.Println(err)
		c.JSON(http.StatusConflict, gin.H{"error": ErrRegisterAlready})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"message": "registered",
		"user":    user,
	})
}

func (s *SessionControl) Connect(c *gin.Context) {
	var raw system.RawUser
	if err := c.BindJSON(&raw); err != nil {
		s.logger.Println(err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": ErrConnect})
		return
	}
	user, err := s.db.Authenticate(c, raw.Username, raw.Password)
	if err != nil {
		s.logger.Println(err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": ErrConnect})
		return
	}
	session := sessions.Default(c)
	session.Clear()
	session.Set("username", user.Name)
	session.Set(id_key, s.getNewId())
	session.Set(auth_key, auth_pass_string)
	if err := session.Save(); err != nil {
		s.logger.Println(err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to save session"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"message": "connected",
		// "id":      id,
	})
}

func (s *SessionControl) Disconnect(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Save()
	c.JSON(http.StatusOK, gin.H{
		"message": "disconnected",
	})
}

func (s *SessionControl) Status(c *gin.Context) {
	session := sessions.Default(c)
	if session.Get(id_key) == nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "disconnected"})
		return
	}

	s.logger.Println("user status requested: ", session.Get(id_key))

	c.JSON(http.StatusOK, gin.H{
		"message": "connected",
	})
}

func (s *SessionControl) GetSession(c *gin.Context) {
	session := sessions.Default(c)
	if session.Get(id_key) == nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "disconnected"})
		return
	}
	s.logger.Println("user session requested: ", session.Get(id_key))
	c.JSON(http.StatusOK, gin.H{
		"message": "connected",
		"id":      session.Get(id_key),
		"user":    session.Get("username"),
	})
}

// user routes
func (s *SessionControl) UserRoutePass(c *gin.Context) {
	session := sessions.Default(c)
	if auth, ok := session.Get(auth_key).(string); ok && auth == auth_pass_string {
		c.Set("id", session.Get("id"))
		c.Set("username", session.Get("username"))
		c.Next() // continue
	} else if c.Request.URL.Path == LOGIN_URL_PATH {
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
