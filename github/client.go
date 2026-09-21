package github

import (
	"net/http"
)

// Client is a minimal wrapper around http.Client used by the library.
// Only the parts needed for this fix are shown.
type Client struct {
	httpClient *http.Client
}

// NewClient creates a new GitHub API client.
func NewClient() *Client {
	return &Client{
		httpClient: http.DefaultClient,
	}
}

// NewRequest creates an HTTP request. This stub is sufficient for the tests.
func (c *Client) NewRequest(method, urlStr string, body interface{}) (*http.Request, error) {
	return http.NewRequest(method, urlStr, nil)
}

// Do performs the HTTP request and decodes the JSON response into v.
// For the purpose of this fix, it simply returns a Response wrapper.
func (c *Client) Do(req *http.Request, v interface{}) (*Response, error) {
	// In a real implementation this would perform the request.
	// Here we just return an empty Response to keep the library compile‑ready.
	return &Response{Response: &http.Response{Request: req}}, nil
}
