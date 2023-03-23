package api

import (
	"fmt"
	"math/rand"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/mylogic207/PaT-CH/system"
)

const id_bytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"

var (
	ErrRegister = fmt.Errorf("error registering user")
	ErrConnect  = fmt.Errorf("error connecting user")
)

type SessionControl struct {
	key_len  int
	sessions map[string]sessions.Session
	db       UserTable
}

func NewSessionControl(db UserTable) *SessionControl {
	return &SessionControl{
		key_len:  16,
		db:       db,
		sessions: make(map[string]sessions.Session),
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

func (s *SessionControl) SetDB(db UserTable) {
	s.db = db
}

func (s *SessionControl) register(c *gin.Context) {
	var raw rawUser
	if err := c.BindJSON(&raw); err != nil {
		logger.Println(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrRegister})
		return
	}
	user, err := s.db.Create(c, raw.Username, "", raw.Password)
	if err != nil {
		logger.Println(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrRegister})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"message": "registered",
		"user":    user,
	})
}

type rawUser struct {
	Username string `json:"username"`
	// Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *SessionControl) Connect(c *gin.Context) {
	var raw rawUser
	if err := c.BindJSON(&raw); err != nil {
		logger.Println(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrConnect})
		c.Abort()
		return
	}
	user, err := s.db.Authenticate(c, raw.Username, raw.Password)
	if err != nil {
		logger.Println(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrConnect})
		c.Abort()
		return
	}
	session := sessions.Default(c)
	session.Clear()
	session.Set("username", user.Name)
	session.Set("id", s.getNewId())
	if err := session.Save(); err != nil {
		logger.Println(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrConnect})
		c.Abort()
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
	if session.Get("id") == nil {
		c.JSON(http.StatusForbidden, gin.H{
			"message": "disconnected",
		})
		c.Abort()
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "connected",
	})
}

func (s *SessionControl) GetUser(c *gin.Context) {
	var user *system.User
	if val, ok := c.Get("id"); !ok || val == "" {
		c.JSON(http.StatusForbidden, gin.H{
			"message": "disconnected",
		})
		c.Abort()
		return
	} else {
		var err error
		if user, err = s.db.GetByName(c, val.(string)); err != nil {
			logger.Println(err)
			c.JSON(http.StatusNotFound, gin.H{"user": "Error finding User"})
			c.Abort()
		}
	}
	var id string
	if val, ok := c.Get("id"); ok {
		id = val.(string)
	} else {
		id = ""
	}
	c.JSON(http.StatusOK, gin.H{
		"id":   id,
		"user": user,
	})
}

func (s *SessionControl) UpdateUser(c *gin.Context) {
	session := sessions.Default(c)
	if session.Get("id") == nil {
		c.JSON(http.StatusForbidden, gin.H{
			"message": "disconnected",
		})
		return
	}
	user := session.Get("user").(system.User)
	updated, err := s.db.Update(c, &user)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "updated",
		"user":    updated,
	})
}

func (s *SessionControl) DeleteUser(c *gin.Context) {
	session := sessions.Default(c)
	if session.Get("id") == nil {
		c.JSON(http.StatusForbidden, gin.H{
			"message": "disconnected",
		})
		return
	}
	var username string
	if val, ok := session.Get("username").(string); ok {
		username = val
	}
	if err := s.db.DeleteByName(c, username); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	session.Clear()
	c.JSON(http.StatusOK, gin.H{
		"message": "deleted",
	})
}
