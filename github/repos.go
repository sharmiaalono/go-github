package github

import (
	"context"
	"fmt"
	"net/http"
)

// RepositoriesService handles communication with the repository related
// methods of the GitHub API.
type RepositoriesService struct {
	client *Client
}

// Repository represents a GitHub repository.
type Repository struct {
	ID          int64  `json:"id,omitempty"`
	NodeID      string `json:"node_id,omitempty"`
	Name        string `json:"name,omitempty"`
	FullName    string `json:"full_name,omitempty"`
	Description string `json:"description,omitempty"`
	Private     bool   `json:"private,omitempty"`
	HTMLURL     string `json:"html_url,omitempty"`
	CloneURL    string `json:"clone_url,omitempty"`
}

// RepositoryListOptions specifies the optional parameters to the
// RepositoriesService.List method.
type RepositoryListOptions struct {
	Visibility  string `url:"visibility,omitempty"`
	Affiliation string `url:"affiliation,omitempty"`
	Type        string `url:"type,omitempty"`
	Sort        string `url:"sort,omitempty"`
	Direction   string `url:"direction,omitempty"`

	ListOptions
}

// List repositories for a user. Passing the empty string will list
// repositories for the authenticated user.
func (s *RepositoriesService) List(ctx context.Context, user string, opts *RepositoryListOptions) ([]*Repository, *Response, error) {
	var u string
	if user != "" {
		u = fmt.Sprintf("users/%v/repos", user)
	} else {
		u = "user/repos"
	}

	u, err := addOptions(u, opts)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest(http.MethodGet, u, nil)
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
