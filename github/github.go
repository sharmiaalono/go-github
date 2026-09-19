package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	defaultBaseURL = "https://api.github.com/"
	userAgent      = "go-github"
)

// Client manages communication with the GitHub API.
type Client struct {
	client    *http.Client
	BaseURL   *url.URL
	UserAgent string
}

// Rate represents the rate limits for the current client.
type Rate struct {
	Limit     int       `json:"limit"`
	Remaining int       `json:"remaining"`
	Reset     Timestamp `json:"reset"`
	Resource  string    `json:"resource"`
}

// Timestamp represents a time that can be unmarshalled from a JSON string or int.
type Timestamp struct {
	time.Time
}

// Response is a GitHub API response. This wraps the standard http.Response
// and provides convenient access to pagination information and rate limits.
type Response struct {
	*http.Response

	NextPage  int
	PrevPage  int
	FirstPage int
	LastPage  int

	PerPage int

	NextPageToken string
	Cursor        string
	Before        string
	After         string

	NextURL  string
	PrevURL  string
	FirstURL string
	LastURL  string

	Rate Rate
}

// NewClient returns a new GitHub API client. If a nil httpClient is
// provided, a default http.Client will be used.
func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	baseURL, _ := url.Parse(defaultBaseURL)
	return &Client{
		client:    httpClient,
		BaseURL:   baseURL,
		UserAgent: userAgent,
	}
}

// NewRequest creates an API request. A relative URL can be provided in urlStr,
// in which case it is resolved relative to the BaseURL of the Client.
func (c *Client) NewRequest(method, urlStr string, body any) (*http.Request, error) {
	u, err := c.BaseURL.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	var buf io.ReadWriter
	if body != nil {
		buf = &bytes.Buffer{}
		enc := json.NewEncoder(buf)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(body); err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, u.String(), buf)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}

	return req, nil
}

// Do sends an API request and returns the API response. The API response is
// JSON decoded and stored in the value pointed to by v, or returned as an
// error if an API error has occurred.
func (c *Client) Do(ctx context.Context, req *http.Request, v any) (*Response, error) {
	if ctx != nil {
		req = req.WithContext(ctx)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	response := newResponse(resp)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return response, fmt.Errorf("github api error: status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	if v != nil {
		if w, ok := v.(io.Writer); ok {
			_, err = io.Copy(w, resp.Body)
		} else {
			err = json.NewDecoder(resp.Body).Decode(v)
		}
		if err != nil && err != io.EOF {
			return response, err
		}
	}

	return response, nil
}

func newResponse(r *http.Response) *Response {
	response := &Response{Response: r}
	parsePagination(response)
	response.Rate = parseRate(r)
	return response
}

func parseRate(r *http.Response) Rate {
	var rate Rate
	if r == nil {
		return rate
	}
	if limit := r.Header.Get("X-RateLimit-Limit"); limit != "" {
		rate.Limit, _ = strconv.Atoi(limit)
	}
	if rem := r.Header.Get("X-RateLimit-Remaining"); rem != "" {
		rate.Remaining, _ = strconv.Atoi(rem)
	}
	if reset := r.Header.Get("X-RateLimit-Reset"); reset != "" {
		if ts, err := strconv.ParseInt(reset, 10, 64); err == nil {
			rate.Reset = Timestamp{Time: time.Unix(ts, 0)}
		}
	}
	rate.Resource = r.Header.Get("X-RateLimit-Resource")
	return rate
}
