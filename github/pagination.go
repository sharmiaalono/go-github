package github

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"

	"github.com/google/go-querystring/query"
)

// ListOptions specifies optional parameters to various List methods that
// support offset pagination. When PerPage is 0, the per_page parameter is
// omitted from request queries, defaulting to the GitHub API standard
// (typically 30 items).
type ListOptions struct {
	Page    int `url:"page,omitempty"`
	PerPage int `url:"per_page,omitempty"`
}

// AddOptions adds the parameters in opts as URL query parameters to s.
// opts must be a struct or pointer to a struct whose fields contain url tags.
func AddOptions(s string, opts any) (string, error) {
	return addOptions(s, opts)
}

func addOptions(s string, opts any) (string, error) {
	if opts == nil {
		return s, nil
	}

	v := reflect.ValueOf(opts)
	if v.Kind() == reflect.Ptr && v.IsNil() {
		return s, nil
	}

	u, err := url.Parse(s)
	if err != nil {
		return s, err
	}

	qs, err := query.Values(opts)
	if err != nil {
		return s, err
	}

	existing := u.Query()
	for k, vals := range existing {
		if _, ok := qs[k]; !ok {
			qs[k] = vals
		}
	}

	u.RawQuery = qs.Encode()
	return u.String(), nil
}

// BuildURL constructs or updates a URL with the given ListOptions, ensuring
// pagination consistency across iterations without corrupting existing query parameters.
func BuildURL(rawURL string, opts *ListOptions) (string, error) {
	return buildURL(rawURL, opts)
}

func buildURL(rawURL string, opts *ListOptions) (string, error) {
	if opts == nil {
		return rawURL, nil
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL, err
	}

	q := u.Query()

	if opts.Page > 0 {
		q.Set("page", strconv.Itoa(opts.Page))
	}

	if opts.PerPage > 0 {
		q.Set("per_page", strconv.Itoa(opts.PerPage))
	} else if opts.PerPage < 0 {
		q.Del("per_page")
	}

	u.RawQuery = q.Encode()
	return u.String(), nil
}

// ParsePagination parses HTTP Link response headers and populates pagination values in Response.
func ParsePagination(r *Response) {
	parsePagination(r)
}

func parsePagination(r *Response) {
	if r == nil || r.Response == nil {
		return
	}

	links, ok := r.Response.Header["Link"]
	if !ok || len(links) == 0 {
		return
	}

	for _, linkHeader := range links {
		for _, link := range strings.Split(linkHeader, ",") {
			segments := strings.Split(strings.TrimSpace(link), ";")
			if len(segments) < 2 {
				continue
			}

			rawURL := strings.TrimSpace(segments[0])
			if !strings.HasPrefix(rawURL, "<") || !strings.HasSuffix(rawURL, ">") {
				continue
			}

			targetURL := rawURL[1 : len(rawURL)-1]
			parseableURL := targetURL
			if !strings.Contains(parseableURL, "://") {
				if strings.HasPrefix(parseableURL, "//") {
					parseableURL = "http:" + parseableURL
				} else if !strings.HasPrefix(parseableURL, "/") {
					parseableURL = "http://" + parseableURL
				}
			}

			parsedURL, err := url.Parse(parseableURL)
			if err != nil {
				continue
			}

			q := parsedURL.Query()

			if ppStr := q.Get("per_page"); ppStr != "" {
				if pp, err := strconv.Atoi(ppStr); err == nil && pp > 0 {
					r.PerPage = pp
				}
			}

			cursor := q.Get("cursor")
			page := q.Get("page")
			since := q.Get("since")
			before := q.Get("before")
			after := q.Get("after")

			if since != "" && page == "" {
				page = since
			}

			for _, segment := range segments[1:] {
				part := strings.TrimSpace(segment)
				if !strings.HasPrefix(part, "rel=") {
					continue
				}
				rel := strings.Trim(strings.TrimPrefix(part, "rel="), `"`)

				switch rel {
				case "next":
					if cursor != "" {
						r.Cursor = cursor
					}
					if pageNum, err := strconv.Atoi(page); err == nil {
						r.NextPage = pageNum
					} else if page != "" {
						r.NextPageToken = page
					}
					r.After = after
					r.NextURL = targetURL
				case "prev":
					if pageNum, err := strconv.Atoi(page); err == nil {
						r.PrevPage = pageNum
					}
					r.Before = before
					r.PrevURL = targetURL
				case "first":
					if pageNum, err := strconv.Atoi(page); err == nil {
						r.FirstPage = pageNum
					}
					r.FirstURL = targetURL
				case "last":
					if pageNum, err := strconv.Atoi(page); err == nil {
						r.LastPage = pageNum
					}
					r.LastURL = targetURL
				}
			}
		}
	}
}

// PopulatePageValues parses HTTP Link response headers to populate pagination values.
func (r *Response) PopulatePageValues() {
	parsePagination(r)
}

// NextPageOptions returns a new ListOptions for retrieving the next page,
// preserving the PerPage configuration from the response.
func (r *Response) NextPageOptions() *ListOptions {
	if r == nil || r.NextPage == 0 {
		return nil
	}
	return &ListOptions{
		Page:    r.NextPage,
		PerPage: r.PerPage,
	}
}

// PrevPageOptions returns a new ListOptions for retrieving the previous page,
// preserving the PerPage configuration from the response.
func (r *Response) PrevPageOptions() *ListOptions {
	if r == nil || r.PrevPage == 0 {
		return nil
	}
	return &ListOptions{
		Page:    r.PrevPage,
		PerPage: r.PerPage,
	}
}

// FirstPageOptions returns a new ListOptions for retrieving the first page,
// preserving the PerPage configuration from the response.
func (r *Response) FirstPageOptions() *ListOptions {
	if r == nil || r.FirstPage == 0 {
		return nil
	}
	return &ListOptions{
		Page:    r.FirstPage,
		PerPage: r.PerPage,
	}
}

// LastPageOptions returns a new ListOptions for retrieving the last page,
// preserving the PerPage configuration from the response.
func (r *Response) LastPageOptions() *ListOptions {
	if r == nil || r.LastPage == 0 {
		return nil
	}
	return &ListOptions{
		Page:    r.LastPage,
		PerPage: r.PerPage,
	}
}

// FollowPage requests a target URL extracted from a pagination response.
func (c *Client) FollowPage(ctx context.Context, pageURL string, v any) (*Response, error) {
	req, err := c.NewRequest(http.MethodGet, pageURL, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(ctx, req, v)
}

// NextPage fetches the next page of results for the given Response.
func (c *Client) NextPage(ctx context.Context, resp *Response, v any) (*Response, error) {
	if resp == nil || (resp.NextURL == "" && resp.NextPage == 0) {
		return nil, fmt.Errorf("no next page available")
	}
	if resp.NextURL != "" {
		return c.FollowPage(ctx, resp.NextURL, v)
	}
	if resp.Request == nil || resp.Request.URL == nil {
		return nil, fmt.Errorf("underlying request is missing")
	}
	nextURL, err := buildURL(resp.Request.URL.String(), resp.NextPageOptions())
	if err != nil {
		return nil, err
	}
	return c.FollowPage(ctx, nextURL, v)
}
