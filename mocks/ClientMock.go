package mocks

import (
	"github.com/yourorg/yourrepo/client"
)

// ClientMock is a mock implementation of the GitHub API client.
type ClientMock struct {
	IssuesFunc func(owner, repo string, opts *client.ListOptions) ([]client.Issue, *client.Response, error)
}

// ListIssues mocks the ListIssues method.
func (m *ClientMock) ListIssues(owner, repo string, opts *client.ListOptions) ([]client.Issue, *client.Response, error) {
	if m.IssuesFunc != nil {
		return m.IssuesFunc(owner, repo, opts)
	}
	return nil, nil, nil
}
