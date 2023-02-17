package api

import (
	"log"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
)

func startTestServer() *Server {
	log.Println("Starting Test Server")
	gin.SetMode(gin.ReleaseMode)
	s := NewServerPreConf(&ApiConfig{
		Host:  "127.0.0.1",
		Port:  2080,
		Redis: false,
	})
	s.Start()
	return s
}

func TestServer(t *testing.T) {
	t.Log("Testing Server Requests")
	server := startTestServer()

	req, err := http.NewRequest("GET", server.Addr("/api/v1/health"), nil)
	if err != nil {
		t.Error(err)
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
	server.Stop()
}

func TestSession(t *testing.T) {
	t.Log("Testing Session Middleware")
	server := startTestServer()
	req, err := http.NewRequest("GET", server.Addr("/api/v1/auth/session"), nil)
	if err != nil {
		t.Error(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
	server.Stop()
}
