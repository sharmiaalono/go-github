package github

import (
	"context"
	"fmt"
	"net/http"
)

// IssueListOptions embeds ListOptions and adds issue‑specific filters.
type IssueListOptions struct {
	ListOptions
	// Additional fields omitted for brevity.
}

// ListIssues lists issues for a repository. The function demonstrates that
// embedding ListOptions works unchanged after the pagination fix.
func (c *Client) ListIssues(ctx context.Context, owner, repo string, opts *IssueListOptions) ([]*Issue, *Response, error) {
	u := fmt.Sprintf("repos/%s/%s/issues", owner, repo)
	u, err := addOptions(u, opts)
	if err != nil {
		return nil, nil, err
	}
	req, err := c.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}
	var issues []*Issue
	resp, err := c.Do(ctx, req, &issues)
	if err != nil {
		return nil, resp, err
	}
	parsedResp, err := parsePagination(resp.Response)
	if err != nil {
		return nil, resp, err
	}
	// Overwrite pagination fields with the parsed values.
	resp.NextPage = parsedResp.NextPage
	resp.PrevPage = parsedResp.PrevPage
	resp.FirstPage = parsedResp.FirstPage
	resp.LastPage = parsedResp.LastPage
	return issues, resp, nil
}
