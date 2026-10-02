// Package github provides a Go client library for accessing the GitHub API.
package github

import (
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
)

// ListOptions specifies the optional parameters to various List methods.
// When PerPage is 0, per_page is omitted from the request query string,
// allowing the GitHub API to apply its standard default (typically 30 items).
type ListOptions struct {
	// Page specifies the page of results to retrieve.
	Page int `url:"page,omitempty"`

	// PerPage specifies the number of results per page.
	// If 0, the per_page parameter is omitted from the URL and GitHub's server default (30) is used.
	PerPage int `url:"per_page,omitempty"`
}

// Response is a GitHub API response. It wraps the standard http.Response
// and provides pagination helper values extracted from the Link header.
type Response struct {
	*http.Response

	// Page values extracted from the Link header relations.
	// A value of 0 indicates the relation was absent.
	NextPage  int
	PrevPage  int
	FirstPage int
	LastPage  int
}

// Client manages communication with the GitHub API.
type Client struct {
	// BaseURL for API requests.
	BaseURL *url.URL

	// HTTPClient used to communicate with the API.
	HTTPClient *http.Client
}

// NewClient returns a new GitHub API client.
func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	baseURL, _ := url.Parse("https://api.github.com/")
	return &Client{
		BaseURL:    baseURL,
		HTTPClient: httpClient,
	}
}

// addOptions adds the parameters in opts as URL query parameters to s.
// opts must be a struct or pointer to a struct containing `url` tags.
// When a field's tag includes `omitempty` and the field holds its zero value (e.g. PerPage: 0),
// the parameter is omitted from the query string.
func addOptions(s string, opts interface{}) (string, error) {
	u, err := url.Parse(s)
	if err != nil {
		return s, err
	}

	if opts == nil {
		return u.String(), nil
	}

	v := reflect.ValueOf(opts)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return u.String(), nil
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return u.String(), nil
	}

	q := u.Query()
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("url")
		if tag == "" || tag == "-" {
			continue
		}

		parts := strings.Split(tag, ",")
		key := parts[0]
		omitempty := false
		for _, opt := range parts[1:] {
			if opt == "omitempty" {
				omitempty = true
				break
			}
		}

		fieldVal := v.Field(i)
		if omitempty && isZero(fieldVal) {
			continue
		}

		switch fieldVal.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			q.Set(key, strconv.FormatInt(fieldVal.Int(), 10))
		case reflect.String:
			q.Set(key, fieldVal.String())
		case reflect.Bool:
			q.Set(key, strconv.FormatBool(fieldVal.Bool()))
		}
	}

	u.RawQuery = q.Encode()
	return u.String(), nil
}

func isZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.String:
		return v.String() == ""
	case reflect.Bool:
		return !v.Bool()
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	default:
		return v.IsZero()
	}
}

// parsePagination parses the RFC 5988 Link header and populates page values
// (NextPage, PrevPage, FirstPage, LastPage) on the Response struct.
// Page numbers are extracted directly from the `page` query parameter of each relation URL,
// regardless of whether `per_page` was explicitly set or omitted by the server.
func parsePagination(resp *Response, header http.Header) {
	if resp == nil || header == nil {
		return
	}

	linkHeader := header.Get("Link")
	if linkHeader == "" {
		return
	}

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

		pageStr := u.Query().Get("page")
		if pageStr == "" {
			continue
		}

		page, err := strconv.Atoi(pageStr)
		if err != nil {
			continue
		}

		switch rel {
		case "next":
			resp.NextPage = page
		case "prev":
			resp.PrevPage = page
		case "first":
			resp.FirstPage = page
		case "last":
			resp.LastPage = page
		}
	}
}

// Do performs an API call and returns a Response with parsed pagination helpers.
func (c *Client) Do(rawurl string, opts interface{}) (*Response, error) {
	u, err := addOptions(rawurl, opts)
	if err != nil {
		return nil, err
	}

	relURL, err := url.Parse(u)
	if err != nil {
		return nil, err
	}

	fullURL := c.BaseURL.ResolveReference(relURL)
	httpReq, err := http.NewRequest("GET", fullURL.String(), nil)
	if err != nil {
		return nil, err
	}

	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}

	resp := &Response{Response: httpResp}
	parsePagination(resp, httpResp.Header)

	return resp, nil
}
