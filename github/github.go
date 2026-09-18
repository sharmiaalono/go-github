package github

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// ListOptions specifies the optional parameters to various List methods that
// support pagination.
//
// PerPage of 0 omits the per_page query parameter so the GitHub API uses its
// default page size (typically 30). Link-header pagination still carries any
// per_page the server includes.
type ListOptions struct {
	// Page for pagination.
	Page int `url:"page,omitempty"`
	// PerPage is the number of items per page. Zero means omit per_page and
	// use the API default.
	PerPage int `url:"per_page,omitempty"`
}

// Response wraps an HTTP response and pagination positions from Link headers.
type Response struct {
	*http.Response

	NextPage  int
	PrevPage  int
	FirstPage int
	LastPage  int

	// PerPage is taken from Link when present (even if ListOptions.PerPage was 0).
	PerPage int
}

// parsePagination fills page fields from an RFC 5988 Link header.
func parsePagination(r *Response) {
	if r == nil || r.Response == nil {
		return
	}
	links := r.Header.Get("Link")
	if links == "" {
		return
	}
	for _, link := range strings.Split(links, ",") {
		link = strings.TrimSpace(link)
		parts := strings.Split(link, ";")
		if len(parts) < 2 {
			continue
		}
		uPart := strings.TrimSpace(parts[0])
		uPart = strings.Trim(uPart, "<>")
		u, err := url.Parse(uPart)
		if err != nil {
			continue
		}
		q := u.Query()
		page, _ := strconv.Atoi(q.Get("page"))
		if pp := q.Get("per_page"); pp != "" {
			if n, err := strconv.Atoi(pp); err == nil && n > 0 {
				r.PerPage = n
			}
		}
		rel := ""
		for _, p := range parts[1:] {
			p = strings.TrimSpace(p)
			if strings.HasPrefix(p, "rel=") {
				rel = strings.Trim(strings.TrimPrefix(p, "rel="), `"`)
			}
		}
		switch rel {
		case "next":
			r.NextPage = page
		case "prev":
			r.PrevPage = page
		case "first":
			r.FirstPage = page
		case "last":
			r.LastPage = page
		}
	}
}

// addOptions appends opts as query parameters onto s.
// Zero-valued PerPage is omitted (omitempty semantics) via a tiny manual encoder.
func addOptions(s string, opts *ListOptions) (string, error) {
	if opts == nil {
		return s, nil
	}
	u, err := url.Parse(s)
	if err != nil {
		return s, err
	}
	q := u.Query()
	if opts.Page != 0 {
		q.Set("page", strconv.Itoa(opts.Page))
	}
	if opts.PerPage != 0 {
		q.Set("per_page", strconv.Itoa(opts.PerPage))
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// NewResponse builds a Response and parses Link pagination.
func NewResponse(r *http.Response) *Response {
	resp := &Response{Response: r}
	parsePagination(resp)
	return resp
}

// FormatLink is a test helper.
func FormatLink(rel string, page, perPage int) string {
	if perPage > 0 {
		return fmt.Sprintf(`<https://api.github.com/items?page=%d&per_page=%d>; rel="%s"`, page, perPage, rel)
	}
	return fmt.Sprintf(`<https://api.github.com/items?page=%d>; rel="%s"`, page, rel)
}
