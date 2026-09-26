package mocks

import (
	"net/http"
	"github.com/example/go-github/client"
)

type ClientMock struct {
	DoListFunc func(urlStr string, opts *client.ListOptions) (*client.Response, *http.Response, error)
}

func (m *ClientMock) DoList(urlStr string, opts *client.ListOptions) (*client.Response, *http.Response, error) {
	if m.DoListFunc != nil {
		return m.DoListFunc(urlStr, opts)
	}
	return nil, nil, nil
}
