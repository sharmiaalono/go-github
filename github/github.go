// Package github contains small, dependency-free helpers for paginated API
// responses. It deliberately keeps the server-provided page size when the
// caller leaves ListOptions.PerPage at its zero value.
package github

import (
	"net/url"
	"strconv"
	"strings"
)

// ListOptions controls a paginated request. PerPage == 0 means that the
// server's default should be used.
type ListOptions struct {
	Page    int
	PerPage int
}

// Response contains pagination links returned by an HTTP API.
type Response struct {
	NextPage  int
	PrevPage  int
	FirstPage int
	LastPage  int
	PerPage   int
}

// ParsePagination parses an RFC 5988 Link header. It returns page numbers and
// the server-provided per_page value, if present. Invalid or missing links are
// ignored instead of turning a successful response into an error.
func ParsePagination(header string) Response {
	var r Response
	for _, part := range strings.Split(header, ",") {
		start, end := strings.IndexByte(part, '<'), strings.IndexByte(part, '>')
		if start < 0 || end <= start {
			continue
		}
		u, err := url.Parse(part[start+1 : end])
		if err != nil {
			continue
		}
		page, err := strconv.Atoi(u.Query().Get("page"))
		if err != nil || page < 1 {
			continue
		}
		if p, err := strconv.Atoi(u.Query().Get("per_page")); err == nil && p > 0 {
			r.PerPage = p
		}
		label := strings.ToLower(part[end+1:])
		if i := strings.Index(label, "rel="); i >= 0 {
			rel := strings.Trim(strings.TrimSpace(label[i+4:]), "\"'")
			switch {
			case strings.Contains(rel, "next"):
				r.NextPage = page
			case strings.Contains(rel, "prev"):
				r.PrevPage = page
			case strings.Contains(rel, "first"):
				r.FirstPage = page
			case strings.Contains(rel, "last"):
				r.LastPage = page
			}
		}
	}
	return r
}

// PageOptions returns options for following a parsed link without losing the
// server's page-size context. A zero page preserves the caller's current page.
func PageOptions(current ListOptions, page int, parsed Response) ListOptions {
	if page > 0 {
		current.Page = page
	}
	if current.PerPage == 0 && parsed.PerPage > 0 {
		current.PerPage = parsed.PerPage
	}
	return current
}
