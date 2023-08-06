package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func randomOffset() uint16 {
	return uint16(rand.Intn(1000))
}

func startTestServer() *Server {
	log.Println("Starting Test Server")
	ctx := context.Background()
	gin.SetMode(gin.ReleaseMode)
	s, err := NewServerWithConf(ctx, NewUserIMDB(), &ApiConfig{
		Host:  "127.0.0.1",
		Port:  3080 + randomOffset(),
		Redis: false,
	})
	if err != nil {
		panic(err)
	}
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
	resp, err := http.DefaultClient.Get(TESTSERVER.Addr("/api/v1/session"))
	fmt.Print("\n")
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
	Message string `json:"message"`
	Id      string `json:"id"`
	User    string `json:"user"`
}

func TestSessionUser(t *testing.T) {
	t.Log("Testing Session Middleware")
	client := http.Client{}
	user := rawUser{
		Username: "test",
		Password: "test123",
	}
	login, err := json.Marshal(user)
	if err != nil {
		t.Error(err)
	}
	register_req, err := http.NewRequest("POST", TESTSERVER.Addr("/api/v1/register"), strings.NewReader(string(login)))
	if err != nil {
		t.Error(err)
	}
	resp, err := client.Do(register_req)
	if err != nil {
		t.Error(err)
	}
	fmt.Print("\n")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected 201, got %d", resp.StatusCode)
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
	if response.Message != "registered" {
		t.Errorf("Expected 'registered', got '%s'", response.Message)
		t.FailNow()
	}
	t.Log("Register Successful")

	resp, err = client.Get(TESTSERVER.Addr("/api/v1/status"))
	if err != nil {
		t.Error(err)
	}
	fmt.Print("\n")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected 401, got %d", resp.StatusCode)
		t.FailNow()
	}

	login_req, err := http.NewRequest("POST", TESTSERVER.Addr("/api/v1/connect"), strings.NewReader(string(login)))
	if err != nil {
		t.Error(err)
	}
	resp, err = client.Do(login_req)
	if err != nil {
		t.Error(err)
	}
	fmt.Print("\n")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected 201, got %d", resp.StatusCode)
		t.FailNow()
	}
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		t.Error(err)
	}
	if len(body) == 0 {
		t.Error("Empty body")
	}
	t.Log(string(body))
	json.Unmarshal([]byte(body), &response)
	if response.Message != "connected" {
		t.Errorf("Expected 'connected', got '%s'", response.Message)
		t.FailNow()
	}
	t.Log("Connect Successful")

	// resp, err = http.DefaultClient.Get(TESTSERVER.Addr("/api/v1/status"))
	// if err != nil {
	// 	t.Error(err)
	// }
	// fmt.Print("\n")
	// defer resp.Body.Close()
	// if resp.StatusCode != http.StatusOK {
	// 	t.Errorf("Expected 200, got %d", resp.StatusCode)
	// 	t.FailNow()
	// }

	resp, err = client.Get(TESTSERVER.Addr("/api/v1/session"))
	if err != nil {
		t.Error(err)
	}
	fmt.Print("\n")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
		t.FailNow()
	}
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		t.Error(err)
	}
	if len(body) == 0 {
		t.Error("Empty body")
	}
	t.Log(string(body))
	json.Unmarshal([]byte(body), &response)
	if response.Message != "connected" {
		t.Errorf("Expected 'connected', got '%s'", response.Message)
	}
	if response.User != user.Username {
		t.Errorf("Expected '%s', got '%s'", user.Username, response.User)
	}
	t.Log("Status Successful")

	delete_user, err := http.NewRequest("DELETE", TESTSERVER.Addr("/api/v1/user"), nil)
	if err != nil {
		t.Error(err)
	}
	resp, err = client.Do(delete_user)
	if err != nil {
		t.Error(err)
	}
	fmt.Print("\n")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		t.Error(err)
	}
	if len(body) == 0 {
		t.Error("Empty body")
	}
	t.Log(string(body))
	json.Unmarshal([]byte(body), &response)
	if response.Message != "deleted" {
		t.Errorf("Expected 'deleted', got '%s'", response.Message)
	}
	t.Log("Delete Successful")
}
