package github

import (
	"context"
	"fmt"
	"net/url"
)

// PullRequestsService exposes pull-request list endpoints.
type PullRequestsService struct {
	client *Client
}

type PullRequest struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
}

type PullRequestListOptions struct {
	ListOptions
	State string `url:"state,omitempty"`
}

// ListByRepo lists pull requests for a repository.
func (service *PullRequestsService) ListByRepo(ctx context.Context, owner, repository string, options *PullRequestListOptions) ([]*PullRequest, *Response, error) {
	path, err := addOptions(fmt.Sprintf("repos/%s/%s/pulls", url.PathEscape(owner), url.PathEscape(repository)), options)
	if err != nil {
		return nil, nil, err
	}
	var pulls []*PullRequest
	response, err := service.client.get(ctx, path, &pulls)
	return pulls, response, err
}
