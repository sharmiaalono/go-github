// Package github provides ListOptions-based pagination helpers that follow
// the GitHub REST API conventions.
//
// Page-size handling: ListOptions.PerPage uses `url:"per_page,omitempty"`,
// so PerPage: 0 omits per_page from the request and the GitHub API serves
// its default page size (typically 30 items). Pagination helpers in this
// package extract page numbers from Link response headers regardless of
// whether per_page was explicitly provided, so following Response.NextPage
// stays consistent across pages (issue #5).
package github

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// ListOptions specifies the optional parameters to paginated List methods
// that support pagination.
//
// When PerPage is 0 it is omitted from the request query string and the
// GitHub API applies its default page size. Callers that walk pages should
// keep reusing the same ListOptions value, updating only Page from
// Response.NextPage — the helpers below never depend on per_page being
// present to resolve page numbers.
type ListOptions struct {
	Page    int `url:"page,omitempty"`
	PerPage int `url:"per_page,omitempty"`
}

// Response holds pagination state parsed from a GitHub API response.
type Response struct {
	// The underlying HTTP response, when available.
	HTTP *http.Response

	// Page numbers parsed from the Link response header. Zero means the
	// relation was absent (e.g. no next page on the last page).
	NextPage  int
	PrevPage  int
	FirstPage int
	LastPage  int
}

// addOptions appends the non-zero ListOptions fields to base as URL query
// parameters. A zero PerPage is omitted so the server default applies.
func addOptions(base string, opts ListOptions) (string, error) {
	u, err := url.Parse(base)
	if err != nil {
		return base, err
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

// PopulatePageValues reads pagination relations out of an HTTP response's
// Link header into the Response. Page numbers are taken from the `page`
// query parameter of each relation URL, independent of whether `per_page`
// was explicitly provided or defaulted server-side (issue #5).
func (r *Response) PopulatePageValues(httpResp *http.Response) {
	if httpResp == nil {
		r.HTTP = httpResp
		return
	}
	r.HTTP = httpResp
	next, prev, first, last := parsePagination(httpResp.Header.Get("Link"))
	r.NextPage = next
	r.PrevPage = prev
	r.FirstPage = first
	r.LastPage = last
}

// parsePagination extracts page numbers for the next/prev/first/last
// relations of an RFC 5988 Link header. Only the `page` parameter of each
// relation URL matters — an explicit, defaulted, or absent `per_page` never
// affects the result.
func parsePagination(linkHeader string) (next, prev, first, last int) {
	for _, part := range strings.Split(linkHeader, ",") {
		segments := strings.Split(strings.TrimSpace(part), ";")
		if len(segments) < 2 {
			continue
		}
		rawURL := strings.Trim(strings.TrimSpace(segments[0]), "<>")
		if rawURL == "" {
			continue
		}
		rel := ""
		for _, attr := range segments[1:] {
			attr = strings.TrimSpace(attr)
			if strings.HasPrefix(attr, "rel=") {
				rel = strings.Trim(strings.TrimPrefix(attr, "rel="), `"`)
			}
		}
		u, err := url.Parse(rawURL)
		if err != nil {
			continue
		}
		page, err := strconv.Atoi(u.Query().Get("page"))
		if err != nil {
			continue
		}
		switch rel {
		case "next":
			next = page
		case "prev":
			prev = page
		case "first":
			first = page
		case "last":
			last = page
		}
	}
	return next, prev, first, last
}
