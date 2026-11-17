package github

import "fmt"

type IssuesService struct{ client *Client }
type Issue struct { Number int `json:"number"`; Title string `json:"title"` }
type IssueListByRepoOptions struct { State string `url:"state,omitempty"`; ListOptions }
func (s *IssuesService) ListByRepo(owner, repo string, opts *IssueListByRepoOptions) ([]*Issue, *Response, error) {
	u, err := addOptions(fmt.Sprintf("repos/%s/%s/issues", owner, repo), opts)
	if err != nil { return nil, nil, err }
	var out []*Issue
	resp, err := s.client.getJSON(u, &out)
	return out, resp, err
}
