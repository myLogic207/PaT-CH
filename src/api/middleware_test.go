package api

import (
	"encoding/json"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func randomOffset() uint16 {
	return uint16(rand.Intn(1000))
}

func startTestServer() *Server {
	log.Println("Starting Test Server")
	gin.SetMode(gin.ReleaseMode)
	s := NewServer(&ApiConfig{
		Host:  "127.0.0.1",
		Port:  2070 + randomOffset(),
		Redis: false,
	})
	s.Start()
	time.Sleep(10 * time.Nanosecond)
	return s
}

var TESTSERVER *Server = startTestServer()

func TestMain(m *testing.M) {
	exit := m.Run()
	time.Sleep(10 * time.Nanosecond)
	TESTSERVER.Stop()
	os.Exit(exit)
}

func TestIndependent(t *testing.T) {
	t.Log("Testing Server Start")

	server := NewServer(&ApiConfig{
		Host:  "localhost",
		Port:  3070 + randomOffset(),
		Redis: false,
	})
	server.Start()
	time.Sleep(time.Duration(100))
	server.Stop()
}

func TestServer(t *testing.T) {
	t.Log("Testing Server Requests")

	resp, err := http.DefaultClient.Get(TESTSERVER.Addr("/api/v1/health"))
	if err != nil {
		t.Error(err)
		return
	}
	t.Log("Request successful")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
	t.Log("Status Successful")
}

func TestSessionNonAuth(t *testing.T) {
	t.Log("Testing Session Middleware")
	resp, err := http.DefaultClient.Get(TESTSERVER.Addr("/api/v1/auth/session"))
	if err != nil {
		t.Error(err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
	t.Log("Status Successful")
}

type SessionResponse struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

func TestSessionAuth(t *testing.T) {
	t.Log("Testing Session Middleware")
	defer TESTSERVER.Stop()
	resp, err := http.DefaultClient.Post(TESTSERVER.Addr("/api/v1/auth/connect"), "application/json", nil)
	if err != nil {
		t.Error(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Error(err)
	}
	if len(body) == 0 {
		t.Error("Empty body")
	}
	t.Log(string(body))
	response := SessionResponse{}
	json.Unmarshal([]byte(body), &response)
	if response.ID == "" {
		t.Error("Empty ID")
	}
	if response.Message != "connected" {
		t.Errorf("Expected 'connected', got '%s'", response.Message)
	}
	t.Log("Session ID:", response.ID)
}
