package github

import "fmt"

type PullRequestsService struct{ client *Client }
type PullRequest struct { Number int `json:"number"`; Title string `json:"title"` }
type PullRequestListOptions struct { State string `url:"state,omitempty"`; ListOptions }
func (s *PullRequestsService) List(owner, repo string, opts *PullRequestListOptions) ([]*PullRequest, *Response, error) {
	u, err := addOptions(fmt.Sprintf("repos/%s/%s/pulls", owner, repo), opts)
	if err != nil { return nil, nil, err }
	var out []*PullRequest
	resp, err := s.client.getJSON(u, &out)
	return out, resp, err
}
