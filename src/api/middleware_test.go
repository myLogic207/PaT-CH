package api

import (
	"net/http"
	"testing"
)

func TestServer(t *testing.T) {
	t.Log("Testing Server Requests")
	server := NewServerPreConf(&ApiConfig{
		Host:  "localhost",
		Port:  2080,
		Redis: false,
	})
	server.Start()
	defer server.Stop()

	req, err := http.NewRequest("GET", server.Addr("/api/v1/health"), nil)
	if err != nil {
		t.Error(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	defer resp.Body.Close()

	req, err = http.NewRequest("GET", server.Addr("/api/v1/auth/session"), nil)
	if err != nil {
		t.Error(err)
	}
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}
