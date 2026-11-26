package github

import (
	"context"
	"fmt"
	"net/url"
)

// IssuesService exposes the issue endpoints used by the regression coverage.
type IssuesService struct {
	client *Client
}

type Issue struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
}

type IssueListByRepoOptions struct {
	ListOptions
	Labels string `url:"labels,omitempty"`
	State  string `url:"state,omitempty"`
}

// ListByRepo lists issues for a repository.
func (service *IssuesService) ListByRepo(ctx context.Context, owner, repository string, options *IssueListByRepoOptions) ([]*Issue, *Response, error) {
	path, err := addOptions(fmt.Sprintf("repos/%s/%s/issues", url.PathEscape(owner), url.PathEscape(repository)), options)
	if err != nil {
		return nil, nil, err
	}
	var issues []*Issue
	response, err := service.client.get(ctx, path, &issues)
	return issues, response, err
}
