package mocks

import (
	"context"
	"github.com/example/go-github/client"
)

type ClientMock struct {
	ListFunc func(ctx context.Context, urlStr string, opt *client.ListOptions) ([]string, *client.Response, error)
}

func (m *ClientMock) List(ctx context.Context, urlStr string, opt *client.ListOptions) ([]string, *client.Response, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, urlStr, opt)
	}
	return nil, nil, nil
}
