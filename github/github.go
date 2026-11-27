// Package github provides a small GitHub API client whose list pagination
// behaves consistently when ListOptions.PerPage is left at its zero value.
//
// When PerPage is 0 the per_page parameter is omitted from the request, so
// the GitHub API applies its own default page size (30, at most 100). The
// page size the server actually used is still recoverable: the URLs in the
// response's Link header carry an explicit per_page parameter, and Response
// exposes it as Response.PerPage. Helpers such as Response.NextPageOptions
// feed that server-selected size back into follow-up requests so that page
// 2, 3, ... are fetched with exactly the same page size as page 1 instead
// of silently dropping the parameter.
package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
)

const (
	defaultBaseURL   = "https://api.github.com/"
	defaultUserAgent = "go-github-client"
)

// Client manages communication with the GitHub API.
type Client struct {
	client *http.Client

	// BaseURL is the base URL for API requests. It must end with a slash.
	BaseURL *url.URL

	// UserAgent is sent with every request.
	UserAgent string

	Issues       *IssuesService
	Repositories *RepositoriesService
	PullRequests *PullRequestsService
}

// NewClient returns a new GitHub API client. If httpClient is nil,
// http.DefaultClient is used.
func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	base, _ := url.Parse(defaultBaseURL)
	c := &Client{client: httpClient, BaseURL: base, UserAgent: defaultUserAgent}
	c.Issues = &IssuesService{client: c}
	c.Repositories = &RepositoriesService{client: c}
	c.PullRequests = &PullRequestsService{client: c}
	return c
}

// ListOptions specifies the optional parameters to various List methods
// that support offset pagination.
type ListOptions struct {
	// Page is the page number of the results to fetch. The zero value
	// asks for the first page.
	Page int `url:"page,omitempty"`

	// PerPage is the number of results to fetch per page. When it is 0,
	// per_page is omitted from the request and the GitHub API applies
	// its default page size. The size the server selected is reported
	// back as Response.PerPage, parsed from the Link header, and
	// Response.NextPageOptions reuses it for follow-up requests so the
	// page size never changes in the middle of a listing.
	PerPage int `url:"per_page,omitempty"`
}

// Response is a GitHub API response. It wraps http.Response and exposes
// the pagination targets advertised in the Link header.
type Response struct {
	*http.Response

	NextPage  int
	PrevPage  int
	FirstPage int
	LastPage  int

	// PerPage is the page size the server actually used, taken from the
	// per_page query parameter of the Link header URLs. It is 0 when the
	// server did not advertise one. It is populated even when the
	// request omitted per_page (ListOptions.PerPage == 0), which is
	// what keeps follow-up page requests consistent with the first one.
	PerPage int
}

func newResponse(r *http.Response) *Response {
	resp := &Response{Response: r}
	resp.populatePageValues(r.Header["Link"])
	return resp
}

// populatePageValues parses RFC 5988 Link headers and records the page
// numbers of the next/prev/first/last relations plus the server-selected
// per_page value. It works whether or not the original request sent an
// explicit per_page parameter.
func (r *Response) populatePageValues(links []string) {
	for _, header := range links {
		for _, segment := range splitLinkHeader(header) {
			target, rel := parseLinkSegment(segment)
			if target == "" || rel == "" {
				continue
			}
			u, err := url.Parse(target)
			if err != nil {
				continue
			}
			q := u.Query()
			page, _ := strconv.Atoi(q.Get("page"))
			switch rel {
			case "next":
				r.NextPage = page
			case "prev":
				r.PrevPage = page
			case "first":
				r.FirstPage = page
			case "last":
				r.LastPage = page
			default:
				continue
			}
			if pp, err := strconv.Atoi(q.Get("per_page")); err == nil && pp > 0 && r.PerPage == 0 {
				r.PerPage = pp
			}
		}
	}
}

// splitLinkHeader splits a Link header value on commas that are not
// inside <...> or inside a quoted string.
func splitLinkHeader(header string) []string {
	var parts []string
	var b strings.Builder
	inAngle, inQuotes := false, false
	for _, ch := range header {
		switch ch {
		case '<':
			if !inQuotes {
				inAngle = true
			}
		case '>':
			if !inQuotes {
				inAngle = false
			}
		case '"':
			if !inAngle {
				inQuotes = !inQuotes
			}
		case ',':
			if !inAngle && !inQuotes {
				parts = append(parts, b.String())
				b.Reset()
				continue
			}
		}
		b.WriteRune(ch)
	}
	if strings.TrimSpace(b.String()) != "" {
		parts = append(parts, b.String())
	}
	return parts
}

// parseLinkSegment extracts the target URL and the rel parameter from a
// single Link header segment such as `<https://...?page=2>; rel="next"`.
func parseLinkSegment(segment string) (target, rel string) {
	segment = strings.TrimSpace(segment)
	if !strings.HasPrefix(segment, "<") {
		return "", ""
	}
	end := strings.Index(segment, ">")
	if end < 0 {
		return "", ""
	}
	target = segment[1:end]
	for _, attr := range strings.Split(segment[end+1:], ";") {
		attr = strings.TrimSpace(attr)
		if strings.HasPrefix(attr, "rel=") {
			rel = strings.Trim(strings.TrimPrefix(attr, "rel="), `"`)
		}
	}
	return target, rel
}

// NextPageOptions returns a copy of opts ready to fetch the next page:
// Page is advanced to Response.NextPage, and when opts.PerPage is 0 the
// page size the server selected for the current page (Response.PerPage)
// is carried over, so follow-up requests keep the active page size
// instead of dropping per_page entirely. It returns nil when there is no
// next page.
func (r *Response) NextPageOptions(opts *ListOptions) *ListOptions {
	if opts == nil || r.NextPage == 0 {
		return nil
	}
	next := *opts
	next.Page = r.NextPage
	if next.PerPage == 0 && r.PerPage > 0 {
		next.PerPage = r.PerPage
	}
	return &next
}

// addOptions adds the parameters of opt, encoded with `url` struct tags,
// to the query string of s. Fields tagged with omitempty are skipped
// when they hold their zero value; in particular a ListOptions with
// PerPage == 0 never produces a "per_page=0" parameter and never erases
// a per_page value already present in s (for example one copied from a
// Link header URL). Anonymous embedded structs (such as an embedded
// ListOptions) are encoded recursively.
func addOptions(s string, opt interface{}) (string, error) {
	v := reflect.ValueOf(opt)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return s, fmt.Errorf("options must be a non-nil pointer to a struct, got %T", opt)
	}
	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return s, fmt.Errorf("options must point to a struct, got %T", opt)
	}
	u, err := url.Parse(s)
	if err != nil {
		return s, err
	}
	q := u.Query()
	if err := encodeStruct(q, v); err != nil {
		return s, err
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func encodeStruct(q url.Values, v reflect.Value) error {
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		fv := v.Field(i)
		if field.Anonymous && fv.Kind() == reflect.Struct {
			if err := encodeStruct(q, fv); err != nil {
				return err
			}
			continue
		}
		if field.PkgPath != "" { // unexported
			continue
		}
		tag := field.Tag.Get("url")
		if tag == "" || tag == "-" {
			continue
		}
		parts := strings.Split(tag, ",")
		name := parts[0]
		omitempty := len(parts) > 1 && parts[1] == "omitempty"
		if omitempty && fv.IsZero() {
			continue
		}
		switch fv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			q.Set(name, strconv.FormatInt(fv.Int(), 10))
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			q.Set(name, strconv.FormatUint(fv.Uint(), 10))
		case reflect.String:
			q.Set(name, fv.String())
		case reflect.Bool:
			q.Set(name, strconv.FormatBool(fv.Bool()))
		case reflect.Slice:
			for j := 0; j < fv.Len(); j++ {
				q.Add(name, fmt.Sprint(fv.Index(j).Interface()))
			}
		default:
			return fmt.Errorf("unsupported type %s for url field %q", fv.Kind(), name)
		}
	}
	return nil
}

// NewRequest creates an API request relative to the client's BaseURL.
func (c *Client) NewRequest(method, urlStr string, body interface{}) (*http.Request, error) {
	rel, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}
	u := c.BaseURL.ResolveReference(rel)

	var buf io.ReadWriter
	if body != nil {
		buf = &bytes.Buffer{}
		if err := json.NewEncoder(buf).Encode(body); err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, u.String(), buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", c.UserAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

// Do sends an API request and returns the API response, decoding the
// JSON body into v when v is not nil.
func (c *Client) Do(ctx context.Context, req *http.Request, v interface{}) (*Response, error) {
	resp, err := c.client.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	response := newResponse(resp)
	if v != nil {
		if err := json.NewDecoder(resp.Body).Decode(v); err != nil && err != io.EOF {
			return response, err
		}
	} else {
		_, _ = io.Copy(io.Discard, resp.Body)
	}
	return response, nil
}
