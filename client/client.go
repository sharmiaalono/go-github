package client

import (
	"net/http"
)

type Client struct {
	BaseURL    string
	HttpClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL:    baseURL,
		HttpClient: http.DefaultClient,
	}
}

func (c *Client) DoList(urlStr string, opts *ListOptions) (*Response, *http.Response, error) {
	urlStr, err := addOptions(urlStr, opts)
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return nil, nil, err
	}

	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}

	response, err := parsePagination(resp)
	if err != nil {
		return response, resp, err
	}

	return response, resp, nil
}
