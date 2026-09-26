package client

import (
	"encoding/json"
	"net/http"
	"net/url"
)

// Issue represents a GitHub issue.
type Issue struct {
	ID     int    `json:"id"`
	Number int    `json:"number"`
	Title  string `json:"title"`
}

// Client is a GitHub API client.
type Client struct {
	// BaseURL is the base URL for API requests.
	BaseURL *url.URL
	// HTTPClient is the HTTP client used for requests.
	HTTPClient *http.Client
}

// NewClient creates a new GitHub API client.
func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	baseURL, _ := url.Parse("https://api.github.com")
	return &Client{
		BaseURL:    baseURL,
		HTTPClient: httpClient,
	}
}

// doRequest executes an HTTP request and returns the response with pagination
// information parsed from the Link header.
func (c *Client) doRequest(req *http.Request) (*Response, error) {
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}

	pagination, err := parsePagination(resp)
	if err != nil {
		resp.Body.Close()
		return nil, err
	}

	return pagination, nil
}

// ListIssues lists issues for a repository with pagination support.
func (c *Client) ListIssues(owner, repo string, opts *ListOptions) ([]Issue, *Response, error) {
	u := c.BaseURL.JoinPath("repos", owner, repo, "issues")

	queryStr, err := addOptions(u.String(), opts)
	if err != nil {
		return nil, nil, err
	}
	u, _ = url.Parse(queryStr)

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	var issues []Issue
	if err := json.NewDecoder(resp.Body).Decode(&issues); err != nil {
		return nil, nil, err
	}

	return issues, resp, nil
}
