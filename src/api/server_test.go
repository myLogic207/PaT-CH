package api

import (
	"fmt"
	"testing"
	"time"
)

func TestNewServer(t *testing.T) {
	t.Log("Testing NewServer")
	server, err := NewServerWithConf(&ApiConfig{
		Host:   "localhost",
		Port:   2070,
		Redis:  false,
		Secure: false,
	})
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	if server == nil {
		t.Error("Server is nil")
	}
	t.Log("Starting server")
	server.Start()
	time.Sleep(1 * time.Millisecond)
	err = server.Stop()
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Println("Stopping finished")
	t.Log("Server stopped successfully")
}
