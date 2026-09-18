package github

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Client is a minimal GitHub API client used to issue paginated list requests
// while preserving ListOptions across page boundaries.
type Client struct {
	BaseURL    *url.URL
	HTTPClient *http.Client
}

// NewClient returns a client that talks to baseURL (for example a mock server).
func NewClient(baseURL string, httpClient *http.Client) (*Client, error) {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	if !strings.HasSuffix(u.Path, "/") {
		u.Path += "/"
	}
	return &Client{BaseURL: u, HTTPClient: httpClient}, nil
}

// List performs a GET against path with opts encoded onto the query string.
func (c *Client) List(ctx context.Context, path string, opts *ListOptions) (*Response, []byte, error) {
	req, err := c.newRequest(ctx, http.MethodGet, path, opts)
	if err != nil {
		return nil, nil, err
	}
	return c.do(req)
}

// NextPage follows resp.NextPage using options that keep PerPage consistent
// even when the caller left it at 0.
func (c *Client) NextPage(ctx context.Context, path string, resp *Response, opts *ListOptions) (*Response, []byte, error) {
	if resp == nil || resp.NextPage == 0 {
		return nil, nil, nil
	}
	current := ListOptions{}
	if opts != nil {
		current = *opts
	}
	next := resp.NextPageOptions(current)
	return c.List(ctx, path, &next)
}

// FollowPage fetches an explicit page, forwarding PerPage from opts or from
// the previous response when opts.PerPage is 0.
func (c *Client) FollowPage(ctx context.Context, path string, page int, resp *Response, opts *ListOptions) (*Response, []byte, error) {
	current := ListOptions{}
	if opts != nil {
		current = *opts
	}
	if resp != nil && current.PerPage == 0 {
		current.PerPage = resp.PerPage
	}
	current.Page = page
	return c.List(ctx, path, &current)
}

func (c *Client) newRequest(ctx context.Context, method, path string, opts *ListOptions) (*http.Request, error) {
	if c == nil || c.BaseURL == nil {
		return nil, fmt.Errorf("github: client is not configured")
	}

	rel, err := url.Parse(strings.TrimPrefix(path, "/"))
	if err != nil {
		return nil, err
	}
	u := c.BaseURL.ResolveReference(rel)
	raw := u.String()
	if opts != nil {
		raw, err = addOptions(raw, opts)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, raw, nil)
	if err != nil {
		return nil, err
	}
	return req, nil
}

func (c *Client) do(req *http.Request) (*Response, []byte, error) {
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	httpResp, err := httpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, nil, err
	}

	resp := newResponse(httpResp)
	if httpResp.StatusCode < 200 || httpResp.StatusCode > 299 {
		return resp, body, fmt.Errorf("github: unexpected status %s", httpResp.Status)
	}
	return resp, body, nil
}
