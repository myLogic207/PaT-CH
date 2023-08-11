package api

import (
	"context"
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/myLogic207/PaT-CH/internal/system"
	"github.com/myLogic207/PaT-CH/pkg/util"
)

var TEST_SERVER *Server

func TestMain(m *testing.M) {
	log.Println("Starting Test Server")
	ctx := context.Background()
	gin.SetMode(gin.ReleaseMode)
	config := util.NewConfig(map[string]interface{}{
		"host": "localhost",
		"port": 12345,
		"redis": map[string]interface{}{
			"use": false,
		},
	}, nil)
	testServer, err := NewServer(ctx, log.Default(), config, system.NewUserIMDB())
	if err != nil {
		panic(err)
	}
	if err := testServer.Start(); err != nil {
		panic(err)
	}
	TEST_SERVER = testServer
	time.Sleep(10 * time.Nanosecond)

	exit := m.Run()

	time.Sleep(10 * time.Nanosecond)
	testServer.Stop()
	os.Exit(exit)
}

func TestServer(t *testing.T) {
	t.Log("Testing Server Requests")

	resp, err := http.DefaultClient.Get(TEST_SERVER.Addr("/api/v1/health"))
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
