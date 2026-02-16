package fetch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetch_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "text/markdown" {
			t.Errorf("Accept header = %q, want text/markdown", r.Header.Get("Accept"))
		}
		w.Write([]byte("# Hello World\nSome content."))
	}))
	defer srv.Close()

	c := NewWithHTTPClient(srv.Client())
	// Use the test server URL as the Jina endpoint by overriding
	// We need a custom approach: set the client to talk to our server
	// by making the fetch URL go through our test server.
	// Since Fetch prepends "https://r.jina.ai/", we test via a transport override.
	c.http.Transport = rewriteTransport{base: srv}

	body, err := c.Fetch(context.Background(), "https://example.com")
	if err != nil {
		t.Fatalf("Fetch() error: %v", err)
	}
	if !strings.Contains(body, "Hello World") {
		t.Errorf("body = %q, want to contain 'Hello World'", body)
	}
}

func TestFetch_Non200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	c := NewWithHTTPClient(srv.Client())
	c.http.Transport = rewriteTransport{base: srv}

	_, err := c.Fetch(context.Background(), "https://example.com")
	if err == nil {
		t.Error("Fetch() should return error for non-200 status")
	}
	if !strings.Contains(err.Error(), "503") {
		t.Errorf("error = %q, want to mention 503", err.Error())
	}
}

func TestFetch_EmptyURL(t *testing.T) {
	c := New()
	_, err := c.Fetch(context.Background(), "")
	if err == nil {
		t.Error("Fetch() should return error for empty URL")
	}
}

func TestFetch_TruncatedAtMaxSize(t *testing.T) {
	bigContent := strings.Repeat("x", MaxBodySize+1000)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(bigContent))
	}))
	defer srv.Close()

	c := NewWithHTTPClient(srv.Client())
	c.http.Transport = rewriteTransport{base: srv}

	body, err := c.Fetch(context.Background(), "https://example.com")
	if err != nil {
		t.Fatalf("Fetch() error: %v", err)
	}
	if len(body) != MaxBodySize {
		t.Errorf("body length = %d, want %d", len(body), MaxBodySize)
	}
}

func TestFetch_InvalidURL_NoScheme(t *testing.T) {
	c := New()
	_, err := c.Fetch(context.Background(), "example.com")
	if err == nil {
		t.Error("Fetch() should return error for URL without scheme")
	}
	if !strings.Contains(err.Error(), "scheme") {
		t.Errorf("error = %q, want to mention scheme", err.Error())
	}
}

// rewriteTransport redirects all requests to the test server, preserving the path.
type rewriteTransport struct {
	base *httptest.Server
}

func (t rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.URL.Scheme = "http"
	req.URL.Host = strings.TrimPrefix(t.base.URL, "http://")
	return http.DefaultTransport.RoundTrip(req)
}
