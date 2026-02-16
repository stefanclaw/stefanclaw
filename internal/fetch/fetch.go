package fetch

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// MaxBodySize is the maximum number of bytes read from a fetch response.
const MaxBodySize = 32 * 1024

// Client fetches web pages via Jina Reader and returns markdown.
type Client struct {
	http *http.Client
}

// New creates a new fetch Client.
func New() *Client {
	return &Client{
		http: &http.Client{Timeout: 30 * time.Second},
	}
}

// NewWithHTTPClient creates a Client with a custom http.Client (for testing).
func NewWithHTTPClient(c *http.Client) *Client {
	return &Client{http: c}
}

// Fetch retrieves the given URL via Jina Reader and returns the content as markdown.
func (c *Client) Fetch(ctx context.Context, rawURL string) (string, error) {
	if rawURL == "" {
		return "", fmt.Errorf("URL is required")
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("URL must have http or https scheme, got %q", parsed.Scheme)
	}

	jinaURL := "https://r.jina.ai/" + rawURL

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, jinaURL, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "text/markdown")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetch failed: HTTP %d", resp.StatusCode)
	}

	limited := io.LimitReader(resp.Body, MaxBodySize+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}

	if len(body) > MaxBodySize {
		body = body[:MaxBodySize]
	}

	return string(body), nil
}
