package api

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestNewServer(t *testing.T) {
	t.Log("Testing NewServer")
	server, err := NewServerWithConf(context.Background(), NewUserIMDB(), &ApiConfig{
		Host:  "localhost",
		Port:  3080,
		Redis: false,
	})
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	if server == nil {
		t.Error("Server is nil")
	}
	t.Log("Starting server")

	if err := server.Start(); !errors.Is(err, ErrStartServer) {
		t.Error(err)
		t.FailNow()
	}
	t.Log("Server started")
	time.Sleep(3 * time.Second)
	if err := server.Stop(); err != nil {
		t.Error(err)
		t.FailNow()
	}
	fmt.Println("Stopping finished")
	t.Log("Server stopped successfully")
}
