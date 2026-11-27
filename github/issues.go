package github

import (
	"context"
	"fmt"
)

// Issue represents a GitHub issue.
type Issue struct {
	Number  int    `json:"number"`
	Title   string `json:"title"`
	State   string `json:"state"`
	HTMLURL string `json:"html_url"`
}

// IssuesService handles communication with the issue related methods of
// the GitHub API.
type IssuesService struct {
	client *Client
}

// IssueListByRepoOptions specifies the optional parameters to the
// IssuesService.ListByRepo method. It embeds ListOptions, so Page and
// PerPage behave exactly as documented there: PerPage == 0 omits
// per_page and defers to the server default.
type IssueListByRepoOptions struct {
	// State filters issues by state: "open", "closed" or "all".
	State string `url:"state,omitempty"`

	ListOptions
}

// ListByRepo lists the issues for the specified repository.
func (s *IssuesService) ListByRepo(ctx context.Context, owner, repo string, opts *IssueListByRepoOptions) ([]*Issue, *Response, error) {
	if opts == nil {
		opts = &IssueListByRepoOptions{}
	}
	u := fmt.Sprintf("repos/%v/%v/issues", owner, repo)
	u, err := addOptions(u, opts)
	if err != nil {
		return nil, nil, err
	}
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}
	var issues []*Issue
	resp, err := s.client.Do(ctx, req, &issues)
	if err != nil {
		return nil, resp, err
	}
	return issues, resp, nil
}
