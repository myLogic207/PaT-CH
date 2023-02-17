package api

import (
	"fmt"
	"testing"
	"time"
)

func TestNewServer(t *testing.T) {
	t.Log("Testing NewServer")
	server := NewServer(&ApiConfig{
		Host:   "localhost",
		Port:   2070,
		Redis:  false,
		Secure: false,
	})
	if server == nil {
		t.Error("Server is nil")
	}
	t.Log("Starting server")
	server.Start()
	time.Sleep(1 * time.Millisecond)
	err := server.Stop()
	if err != nil {
		t.Error(err)
	}
	fmt.Println("Stopping finished")
	t.Log("Server stopped successfully")
}
