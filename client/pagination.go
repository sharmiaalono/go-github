package client

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/google/go-querystring/query"
)

// ListOptions specifies the optional parameters to various List methods that
// support pagination.
type ListOptions struct {
	Page    int `url:"page,omitempty"`
	PerPage int `url:"per_page,omitempty"`
}

// Response represents a GitHub API response with pagination information.
type Response struct {
	*http.Response
	NextPage  int
	PrevPage  int
	FirstPage int
	LastPage  int
}

// NextPage returns the NextPage field value.
func (r *Response) NextPageNum() int { return r.NextPage }

// PrevPage returns the PrevPage field value.
func (r *Response) PrevPageNum() int { return r.PrevPage }

// FirstPage returns the FirstPage field value.
func (r *Response) FirstPageNum() int { return r.FirstPage }

// LastPage returns the LastPage field value.
func (r *Response) LastPageNum() int { return r.LastPage }

// addOptions adds the parameters in opts as query parameters to the baseURL.
// It removes any per_page parameter with value "0" to enforce true omitempty
// behavior, since go-querystring may still encode zero-value ints in some cases.
func addOptions(baseURL string, opts interface{}) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}

	qs, err := query.Values(opts)
	if err != nil {
		return "", err
	}

	// Remove per_page if it's "0" or empty to enforce omitempty behavior
	if v := qs.Get("per_page"); v == "0" || v == "" {
		qs.Del("per_page")
	}

	// Merge with existing query parameters
	existing := u.Query()
	for k, v := range qs {
		for _, val := range v {
			existing.Add(k, val)
		}
	}
	u.RawQuery = existing.Encode()

	return u.String(), nil
}

type parsedLink struct {
	url *url.URL
	rel string
}

// parsePagination parses the Link header of a GitHub API response and
// extracts pagination information. It ignores any per_page parameter in the
// Link header and only extracts page numbers.
func parsePagination(resp *http.Response) (*Response, error) {
	r := &Response{Response: resp}

	linkHeader := resp.Header.Get("Link")
	if linkHeader == "" {
		return r, nil
	}

	links := parseLinkHeader(linkHeader)
	for _, link := range links {
		pageStr := link.url.Query().Get("page")
		if pageStr == "" {
			continue
		}
		page, err := strconv.Atoi(pageStr)
		if err != nil {
			continue
		}
		switch link.rel {
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

// parseLinkHeader parses a Link header value into a slice of parsedLinks.
func parseLinkHeader(header string) []parsedLink {
	var links []parsedLink
	parts := strings.Split(header, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		// Link header format: <url>; rel="relation"
		idx := strings.Index(part, ";")
		if idx != -1 {
			urlPart := strings.TrimSpace(part[:idx])
			urlPart = strings.TrimPrefix(urlPart, "<")
			urlPart = strings.TrimSuffix(urlPart, ">")
			u, err := url.Parse(urlPart)

			relPart := strings.TrimSpace(part[idx+1:])
			rel := ""
			if strings.HasPrefix(relPart, "rel=") {
				rel = strings.TrimPrefix(relPart, "rel=")
				rel = strings.Trim(rel, "\"'")
			}

			if err == nil {
				links = append(links, parsedLink{url: u, rel: rel})
			}
		}
	}
	return links
}
