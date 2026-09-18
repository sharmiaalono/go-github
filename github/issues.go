package github

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// IssuesService handles communication with the issue related methods
// of the GitHub API.
type IssuesService struct {
	client *Client
}

// Issue represents a GitHub issue on a repository.
type Issue struct {
	ID        int64      `json:"id,omitempty"`
	Number    int        `json:"number,omitempty"`
	State     string     `json:"state,omitempty"`
	Title     string     `json:"title,omitempty"`
	Body      string     `json:"body,omitempty"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
	ClosedAt  *time.Time `json:"closed_at,omitempty"`
	HTMLURL   string     `json:"html_url,omitempty"`
}

// IssueListByRepoOptions specifies the optional parameters to the
// IssuesService.ListByRepo method.
type IssueListByRepoOptions struct {
	Milestone string    `url:"milestone,omitempty"`
	State     string    `url:"state,omitempty"`
	Assignee  string    `url:"assignee,omitempty"`
	Creator   string    `url:"creator,omitempty"`
	Mentioned string    `url:"mentioned,omitempty"`
	Labels    []string  `url:"labels,comma,omitempty"`
	Sort      string    `url:"sort,omitempty"`
	Direction string    `url:"direction,omitempty"`
	Since     time.Time `url:"since,omitempty"`

	ListOptions
}

// ListByRepo lists the issues for a specified repository.
func (s *IssuesService) ListByRepo(ctx context.Context, owner, repo string, opts *IssueListByRepoOptions) ([]*Issue, *Response, error) {
	u := fmt.Sprintf("repos/%v/%v/issues", owner, repo)
	u, err := addOptions(u, opts)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest(http.MethodGet, u, nil)
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
