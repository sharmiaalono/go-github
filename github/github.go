package github

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/google/go-querystring/query"
)

// ListOptions specifies the optional parameters to various List methods that
// support pagination.
//
// The zero value for PerPage means "use the GitHub API default" (normally 30).
// The `omitempty` tag ensures that a zero value is not sent to the API, which
// would otherwise be interpreted as a request for zero items.
type ListOptions struct {
	// Page of results to retrieve.
	Page int `url:"page,omitempty"`

	// Number of results per page (0 means use the API default).
	PerPage int `url:"per_page,omitempty"`
}

// Response wraps the standard http.Response returned from the GitHub API.
// It also provides pagination helpers that are populated from the HTTP
// Link header.
type Response struct {
	*http.Response

	// Pagination fields populated by parsePagination.
	NextPage  int
	PrevPage  int
	FirstPage int
	LastPage  int
}

// addOptions adds the parameters in opt to the URL string s. opt must be a
// struct whose fields are tagged with `url:"name,omitempty"`.
// If opt is nil, s is returned unchanged.
func addOptions(s string, opt interface{}) (string, error) {
	if opt == nil {
		return s, nil
	}
	v, err := query.Values(opt)
	if err != nil {
		return s, err
	}
	// If there are no query parameters, return the original string.
	if len(v) == 0 {
		return s, nil
	}
	u, err := url.Parse(s)
	if err != nil {
		return s, err
	}
	// Preserve any existing query parameters.
	q := u.Query()
	for k, vals := range v {
		for _, val := range vals {
			q.Add(k, val)
		}
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// parsePagination extracts pagination information from the HTTP Link header.
// It tolerates the absence of the per_page query parameter (which is the case
// when ListOptions.PerPage is 0 and the API defaults are used).
func parsePagination(resp *http.Response) (*Response, error) {
	r := &Response{Response: resp}
	if resp == nil {
		return r, nil
	}
	linkHeader := resp.Header.Get("Link")
	if linkHeader == "" {
		return r, nil
	}
	// The Link header can contain multiple comma‑separated links.
	parts := strings.Split(linkHeader, ",")
	for _, p := range parts {
		// Example: <https://api.github.com/resource?page=2>; rel="next"
		section := strings.Split(strings.TrimSpace(p), ";")
		if len(section) < 2 {
			continue
		}
		urlPart := strings.Trim(section[0], "<>")
		relPart := ""
		for _, s := range section[1:] {
			if strings.Contains(s, "rel=") {
				relPart = strings.Trim(strings.SplitN(s, "=", 2)[1], `"`)
				break
			}
		}
		if relPart == "" {
			continue
		}
		u, err := url.Parse(urlPart)
		if err != nil {
			continue
		}
		q := u.Query()
		pageStr := q.Get("page")
		if pageStr == "" {
			// Some GitHub responses omit the page parameter for the "first"
			// link when it points to page 1. Treat it as page 1.
			if relPart == "first" {
				pageStr = "1"
			} else {
				continue
			}
		}
		page, err := strconv.Atoi(pageStr)
		if err != nil {
			continue
		}
		switch relPart {
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
	return r, nil
}
