package github

import (
	"context"
	"fmt"
	"net/url"
)

// RepositoriesService exposes repository list endpoints.
type RepositoriesService struct {
	client *Client
}

type Repository struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type RepositoryListOptions struct {
	ListOptions
	Type string `url:"type,omitempty"`
}

// ListByUser lists public repositories for a user.
func (service *RepositoriesService) ListByUser(ctx context.Context, user string, options *RepositoryListOptions) ([]*Repository, *Response, error) {
	path, err := addOptions(fmt.Sprintf("users/%s/repos", url.PathEscape(user)), options)
	if err != nil {
		return nil, nil, err
	}
	var repositories []*Repository
	response, err := service.client.get(ctx, path, &repositories)
	return repositories, response, err
}
