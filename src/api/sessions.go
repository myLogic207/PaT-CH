package api

import (
	"math/rand"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

const id_bytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"

type SessionControl struct {
	key_len  int
	sessions map[string]sessions.Session
}

func NewSessionControl() *SessionControl {
	return &SessionControl{
		key_len: 16,
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

func (s SessionControl) Connect(c *gin.Context) {
	session := sessions.Default(c)
	id := s.getNewId()
	session.Set("id", id)
	session.Save()
	c.JSON(http.StatusOK, gin.H{
		"message": "User Sign in successfully",
		"id":      id,
	})
}

func (s SessionControl) Disonnect(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Save()
	c.JSON(http.StatusOK, gin.H{
		"message": "User Sign out successfully",
	})
}
