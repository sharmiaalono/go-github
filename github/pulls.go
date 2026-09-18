package github

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// PullRequestsService handles communication with the pull request related
// methods of the GitHub API.
type PullRequestsService struct {
	client *Client
}

// PullRequest represents a GitHub pull request on a repository.
type PullRequest struct {
	ID        int64      `json:"id,omitempty"`
	Number    int        `json:"number,omitempty"`
	State     string     `json:"state,omitempty"`
	Title     string     `json:"title,omitempty"`
	Body      string     `json:"body,omitempty"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
	ClosedAt  *time.Time `json:"closed_at,omitempty"`
	MergedAt  *time.Time `json:"merged_at,omitempty"`
	HTMLURL   string     `json:"html_url,omitempty"`
}

// PullRequestListOptions specifies the optional parameters to the
// PullRequestsService.List method.
type PullRequestListOptions struct {
	State     string `url:"state,omitempty"`
	Head      string `url:"head,omitempty"`
	Base      string `url:"base,omitempty"`
	Sort      string `url:"sort,omitempty"`
	Direction string `url:"direction,omitempty"`

	ListOptions
}

// List lists the pull requests for a specified repository.
func (s *PullRequestsService) List(ctx context.Context, owner, repo string, opts *PullRequestListOptions) ([]*PullRequest, *Response, error) {
	u := fmt.Sprintf("repos/%v/%v/pulls", owner, repo)
	u, err := addOptions(u, opts)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, nil, err
	}

	var pulls []*PullRequest
	resp, err := s.client.Do(ctx, req, &pulls)
	if err != nil {
		return nil, resp, err
	}

	return pulls, resp, nil
}
