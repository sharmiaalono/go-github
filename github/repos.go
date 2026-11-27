package github

import (
	"context"
	"fmt"
)

// Repository represents a GitHub repository.
type Repository struct {
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	HTMLURL  string `json:"html_url"`
}

// RepositoriesService handles communication with the repository related
// methods of the GitHub API.
type RepositoriesService struct {
	client *Client
}

// RepositoryListOptions specifies the optional parameters to the
// RepositoriesService.List method. It embeds ListOptions, so Page and
// PerPage behave exactly as documented there: PerPage == 0 omits
// per_page and defers to the server default.
type RepositoryListOptions struct {
	// Type filters repositories by type: "all", "owner" or "member".
	Type string `url:"type,omitempty"`

	ListOptions
}

// List lists the repositories for the specified user.
func (s *RepositoriesService) List(ctx context.Context, user string, opts *RepositoryListOptions) ([]*Repository, *Response, error) {
	if opts == nil {
		opts = &RepositoryListOptions{}
	}
	u := fmt.Sprintf("users/%v/repos", user)
	u, err := addOptions(u, opts)
	if err != nil {
		return nil, nil, err
	}
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}
	var repos []*Repository
	resp, err := s.client.Do(ctx, req, &repos)
	if err != nil {
		return nil, resp, err
	}
	return repos, resp, nil
}
