package ollama

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDetect_OllamaRunning(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"models":[]}`))
	}))
	defer srv.Close()

	err := Detect(context.Background(), srv.URL)
	if err != nil {
		t.Errorf("Detect() error: %v", err)
	}
}

func TestDetect_OllamaNotRunning(t *testing.T) {
	err := Detect(context.Background(), "http://127.0.0.1:1")
	if err == nil {
		t.Error("Detect() should return error for unreachable server")
	}
}
