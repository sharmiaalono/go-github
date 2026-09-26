package client

import (
	"context"
	"net/http"
)

type Client struct {
	BaseURL    string
	HttpClient *http.Client
}

func NewClient() *Client {
	return &Client{
		HttpClient: http.DefaultClient,
	}
	}

func (c *Client) List(ctx context.Context, urlStr string, opt *ListOptions) ([]string, *Response, error) {
	u, err := addOptions(urlStr, opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, nil, err
	}

	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	pResp, err := parsePagination(resp)
	if err != nil {
		return nil, nil, err
	}

	return nil, pResp, nil
}
