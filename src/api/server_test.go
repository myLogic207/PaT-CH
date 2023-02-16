package api

import (
	"testing"
	"time"
)

func TestNewServer(t *testing.T) {
	t.Log("Testing NewServer")
	server := NewServerPreConf(&ApiConfig{
		Host:  "localhost",
		Port:  2070,
		Redis: false,
	})
	if server == nil {
		t.Error("Server is nil")
	}
	t.Log("Starting server")
	server.Start()
	time.Sleep(1 * time.Second)
	err := server.Stop()
	if err != nil {
		t.Error(err)
	}
	t.Log("Server stopped successfully")
}
