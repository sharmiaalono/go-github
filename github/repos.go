package github

import "fmt"

type RepositoriesService struct{ client *Client }
type Repository struct { ID int64 `json:"id"`; Name string `json:"name"` }
type RepositoryListOptions struct { Type string `url:"type,omitempty"`; ListOptions }
func (s *RepositoriesService) List(user string, opts *RepositoryListOptions) ([]*Repository, *Response, error) {
	path := "user/repos"
	if user != "" { path = fmt.Sprintf("users/%s/repos", user) }
	u, err := addOptions(path, opts)
	if err != nil { return nil, nil, err }
	var out []*Repository
	resp, err := s.client.getJSON(u, &out)
	return out, resp, err
}
