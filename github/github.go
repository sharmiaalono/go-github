package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
)

// DefaultPerPage is the GitHub API standard page size used when ListOptions.PerPage is 0.
const DefaultPerPage = 30

// ListOptions specifies optional pagination parameters to various List methods.
// When PerPage is 0, per_page is omitted from outgoing query strings so the GitHub API
// applies its default page size (30), while Link header parsing and NextPage helpers
// preserve any server-supplied per_page parameter across subsequent requests.
type ListOptions struct {
	Page    int `url:"page,omitempty"`
	PerPage int `url:"per_page,omitempty"`
}

// EffectivePerPage returns PerPage if > 0, or DefaultPerPage (30) when PerPage <= 0.
func (o ListOptions) EffectivePerPage() int {
	if o.PerPage > 0 {
		return o.PerPage
	}
	return DefaultPerPage
}

// Response wraps *http.Response and holds parsed pagination page numbers.
type Response struct {
	*http.Response
	NextPage  int
	PrevPage  int
	FirstPage int
	LastPage  int
	PerPage   int
}

// NewResponse creates a new Response for the provided http.Response.
func NewResponse(r *http.Response) *Response {
	response := &Response{Response: r}
	response.populatePageValues()
	return response
}

func (r *Response) populatePageValues() {
	if r == nil || r.Response == nil {
		return
	}
	if r.Request != nil && r.Request.URL != nil {
		if pp, err := strconv.Atoi(r.Request.URL.Query().Get("per_page")); err == nil && pp > 0 {
			r.PerPage = pp
		}
	}
	next, prev, first, last, linkPP := parsePagination(r.Header["Link"])
	r.NextPage = next
	r.PrevPage = prev
	r.FirstPage = first
	r.LastPage = last
	if linkPP > 0 {
		r.PerPage = linkPP
	}
	if r.PerPage == 0 {
		r.PerPage = DefaultPerPage
	}
}

func parsePagination(links []string) (next, prev, first, last, perPage int) {
	for _, headerVal := range links {
		for _, chunk := range strings.Split(headerVal, ",") {
			segments := strings.Split(strings.TrimSpace(chunk), ";")
			if len(segments) < 2 {
				continue
			}
			raw := strings.TrimSpace(segments[0])
			if !strings.HasPrefix(raw, "<") || !strings.HasSuffix(raw, ">") {
				continue
			}
			u, err := url.Parse(raw[1 : len(raw)-1])
			if err != nil {
				continue
			}
			q := u.Query()
			if ppStr := q.Get("per_page"); ppStr != "" {
				if pp, err := strconv.Atoi(ppStr); err == nil && pp > 0 {
					perPage = pp
				}
			}
			pageStr := q.Get("page")
			if pageStr == "" {
				continue
			}
			page, err := strconv.Atoi(pageStr)
			if err != nil || page <= 0 {
				continue
			}
			for _, seg := range segments[1:] {
				kv := strings.SplitN(strings.TrimSpace(seg), "=", 2)
				if len(kv) != 2 || strings.ToLower(strings.TrimSpace(kv[0])) != "rel" {
					continue
				}
				rel := strings.Trim(strings.TrimSpace(kv[1]), `"'`)
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
		}
	}
	return
}

func (r *Response) NextPageOptions(current ListOptions) (ListOptions, bool) {
	if r == nil || r.NextPage == 0 {
		return current, false
	}
	next := ListOptions{Page: r.NextPage, PerPage: current.PerPage}
	if next.PerPage == 0 && r.PerPage > 0 && r.PerPage != DefaultPerPage {
		next.PerPage = r.PerPage
	}
	return next, true
}

func encodeStructFields(v reflect.Value, q url.Values) error {
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil
	}
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		sf := t.Field(i)
		fv := v.Field(i)
		if sf.Anonymous {
			if err := encodeStructFields(fv, q); err != nil {
				return err
			}
			continue
		}
		tag := sf.Tag.Get("url")
		if tag == "" || tag == "-" {
			continue
		}
		parts := strings.Split(tag, ",")
		name := parts[0]
		omitEmpty := len(parts) > 1 && parts[1] == "omitempty"
		switch fv.Kind() {
		case reflect.Int, reflect.Int64:
			iv := fv.Int()
			if iv < 0 && (name == "per_page" || name == "page") {
				return fmt.Errorf("invalid negative %s: %d", name, iv)
			}
			if iv == 0 && omitEmpty {
				if name == "page" {
					q.Del("page")
				}
				continue
			}
			q.Set(name, strconv.FormatInt(iv, 10))
		case reflect.String:
			sv := fv.String()
			if sv == "" && omitEmpty {
				continue
			}
			q.Set(name, sv)
		}
	}
	return nil
}

func addOptions(s string, opt any) (string, error) {
	u, err := url.Parse(s)
	if err != nil {
		return s, err
	}
	if opt == nil {
		return u.String(), nil
	}
	rv := reflect.ValueOf(opt)
	if rv.Kind() == reflect.Ptr && rv.IsNil() {
		return u.String(), nil
	}
	q := u.Query()
	if err := encodeStructFields(rv, q); err != nil {
		return s, err
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

type Client struct {
	client       *http.Client
	BaseURL      *url.URL
	Issues       *IssuesService
	Repositories *RepositoriesService
	PullRequests *PullRequestsService
}

func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	baseURL, _ := url.Parse("https://api.github.com/")
	c := &Client{client: httpClient, BaseURL: baseURL}
	c.Issues = &IssuesService{client: c}
	c.Repositories = &RepositoriesService{client: c}
	c.PullRequests = &PullRequestsService{client: c}
	return c
}

func (c *Client) getJSON(relURL string, v any) (*Response, error) {
	rel, err := url.Parse(relURL)
	if err != nil {
		return nil, err
	}
	u := c.BaseURL.ResolveReference(rel)
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	httpResp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()
	resp := NewResponse(httpResp)
	if v != nil {
		if err := json.NewDecoder(httpResp.Body).Decode(v); err != nil {
			return resp, err
		}
	}
	return resp, nil
}
