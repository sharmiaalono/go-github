package github

import (
	"context"
	"fmt"
)

// PullRequest represents a GitHub pull request.
type PullRequest struct {
	Number  int    `json:"number"`
	Title   string `json:"title"`
	State   string `json:"state"`
	HTMLURL string `json:"html_url"`
}

// PullRequestsService handles communication with the pull request
// related methods of the GitHub API.
type PullRequestsService struct {
	client *Client
}

// PullRequestListOptions specifies the optional parameters to the
// PullRequestsService.List method. It embeds ListOptions, so Page and
// PerPage behave exactly as documented there: PerPage == 0 omits
// per_page and defers to the server default.
type PullRequestListOptions struct {
	// State filters pull requests by state: "open", "closed" or "all".
	State string `url:"state,omitempty"`

	ListOptions
}

// List lists the pull requests for the specified repository.
func (s *PullRequestsService) List(ctx context.Context, owner, repo string, opts *PullRequestListOptions) ([]*PullRequest, *Response, error) {
	if opts == nil {
		opts = &PullRequestListOptions{}
	}
	u := fmt.Sprintf("repos/%v/%v/pulls", owner, repo)
	u, err := addOptions(u, opts)
	if err != nil {
		return nil, nil, err
	}
	req, err := s.client.NewRequest("GET", u, nil)
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
