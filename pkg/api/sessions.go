package api

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

const id = "unique_user_identifier"

const id_bytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"

var (
	ErrRegister        = fmt.Errorf("error registering user")
	ErrRegisterAlready = fmt.Errorf("user already exists")
	ErrConnect         = fmt.Errorf("error connecting user")
)

type SessionControl struct {
	key_len  int
	sessions map[string]sessions.Session
	db       UserTable
	logger   *log.Logger
}

type rawUser struct {
	Username string `json:"username"`
	// Email    string `json:"email"`
	Password string `json:"password"`
}

func NewSessionControl(db UserTable, logger *log.Logger) *SessionControl {
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

func (s *SessionControl) SetDB(db UserTable) {
	s.db = db
}

func (s *SessionControl) register(c *gin.Context) {
	var raw rawUser
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
	var raw rawUser
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
	session.Set(id, s.getNewId())
	session.Set("status", "authorized")
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
