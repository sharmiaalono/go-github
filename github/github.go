// Package github provides a small GitHub API client surface for pagination
// and the list endpoints used by this repository's bounty regression tests.
package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
)

const (
	defaultBaseURL = "https://api.github.com/"
	defaultMediaV3 = "application/vnd.github.v3+json"
	DefaultPerPage = 30
)

// ListOptions contains the query parameters shared by paginated endpoints.
//
// PerPage is omitted from outgoing requests when it is zero, allowing GitHub
// to apply its default page size. Link-header parsing still records the page
// size selected by the server so callers can preserve it on later requests.
type ListOptions struct {
	Page    int `url:"page,omitempty"`
	PerPage int `url:"per_page,omitempty"`
}

// Response wraps http.Response and exposes the page numbers in its Link
// header. A zero page field means the corresponding relation was absent.
type Response struct {
	*http.Response

	NextPage  int
	PrevPage  int
	FirstPage int
	LastPage  int

	// PerPage is the effective page size from the Link header, the request, or
	// the GitHub API default, in that order.
	PerPage int
}

// Client is a minimal GitHub API client. Services are initialized by
// NewClient and may be replaced by tests when a custom base URL is needed.
type Client struct {
	httpClient *http.Client
	BaseURL    *url.URL
	UserAgent  string

	Issues       *IssuesService
	Repositories *RepositoriesService
	PullRequests *PullRequestsService
}

// NewClient creates a GitHub API client. A nil HTTP client uses
// http.DefaultClient.
func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	base, _ := url.Parse(defaultBaseURL)
	client := &Client{
		httpClient: httpClient,
		BaseURL:    base,
		UserAgent:  "go-github-bounty-client",
	}
	client.Issues = &IssuesService{client: client}
	client.Repositories = &RepositoriesService{client: client}
	client.PullRequests = &PullRequestsService{client: client}
	return client
}

// NewRequest creates an API request relative to BaseURL.
func (c *Client) NewRequest(method, relativePath string, body io.Reader) (*http.Request, error) {
	if c == nil || c.BaseURL == nil {
		return nil, errors.New("github: client base URL is not configured")
	}
	relative, err := url.Parse(relativePath)
	if err != nil {
		return nil, err
	}
	absolute := c.BaseURL.ResolveReference(relative)
	request, err := http.NewRequest(method, absolute.String(), body)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", defaultMediaV3)
	if c.UserAgent != "" {
		request.Header.Set("User-Agent", c.UserAgent)
	}
	return request, nil
}

// Do executes a request, parses pagination metadata, and JSON-decodes the
// body into value when value is non-nil.
func (c *Client) Do(request *http.Request, value any) (*Response, error) {
	if request == nil {
		return nil, errors.New("github: nil request")
	}
	httpResponse, err := c.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer httpResponse.Body.Close()

	response := newResponse(httpResponse)
	if httpResponse.StatusCode < 200 || httpResponse.StatusCode >= 300 {
		data, _ := io.ReadAll(httpResponse.Body)
		return response, &ErrorResponse{
			Response: httpResponse,
			Message:  strings.TrimSpace(string(data)),
		}
	}
	if value != nil {
		if err := json.NewDecoder(httpResponse.Body).Decode(value); err != nil {
			return response, err
		}
	}
	return response, nil
}

func (c *Client) get(ctx context.Context, relativePath string, value any) (*Response, error) {
	request, err := c.NewRequest(http.MethodGet, relativePath, nil)
	if err != nil {
		return nil, err
	}
	if ctx != nil {
		request = request.WithContext(ctx)
	}
	return c.Do(request, value)
}

// NextPageOptions returns the caller's typed options with the next page
// number applied. Callers should invoke it after each page:
//
//	next, ok := response.NextPageOptions(options)
//	if ok {
//		options = next
//	}
func (r *Response) NextPageOptions(options ListOptions) (ListOptions, bool) {
	if r == nil || r.NextPage <= 0 {
		return options, false
	}
	options.Page = r.NextPage
	if options.PerPage == 0 && r.PerPage > 0 && r.PerPage != DefaultPerPage {
		options.PerPage = r.PerPage
	}
	return options, true
}

func newResponse(httpResponse *http.Response) *Response {
	response := &Response{Response: httpResponse}
	if httpResponse == nil {
		return response
	}

	pagination := parseLinkHeader(httpResponse.Header.Values("Link"))
	response.NextPage = pagination.next
	response.PrevPage = pagination.prev
	response.FirstPage = pagination.first
	response.LastPage = pagination.last
	response.PerPage = pagination.perPage

	if response.PerPage == 0 && httpResponse.Request != nil && httpResponse.Request.URL != nil {
		if pageSize, err := strconv.Atoi(httpResponse.Request.URL.Query().Get("per_page")); err == nil && pageSize > 0 {
			response.PerPage = pageSize
		}
	}
	if response.PerPage == 0 {
		response.PerPage = DefaultPerPage
	}
	return response
}

type paginationLinks struct {
	next    int
	prev    int
	first   int
	last    int
	perPage int
}

// parseLinkHeader parses RFC 5988 Link headers. Commas inside angle brackets
// or quoted relation values do not split a link entry.
func parseLinkHeader(values []string) paginationLinks {
	var result paginationLinks
	for _, value := range values {
		for _, entry := range splitLinkEntries(value) {
			segments := strings.Split(entry, ";")
			if len(segments) < 2 {
				continue
			}
			target := strings.TrimSpace(segments[0])
			if len(target) < 2 || target[0] != '<' || target[len(target)-1] != '>' {
				continue
			}
			linkURL, err := url.Parse(target[1 : len(target)-1])
			if err != nil {
				continue
			}
			query := linkURL.Query()
			page, err := positiveInt(query.Get("page"))
			if err != nil {
				continue
			}
			if perPage, err := positiveInt(query.Get("per_page")); err == nil && result.perPage == 0 {
				result.perPage = perPage
			}
			for _, parameter := range segments[1:] {
				key, value, ok := strings.Cut(strings.TrimSpace(parameter), "=")
				if !ok || !strings.EqualFold(strings.TrimSpace(key), "rel") {
					continue
				}
				for _, relation := range strings.Fields(strings.Trim(strings.TrimSpace(value), `"'`)) {
					switch relation {
					case "next":
						result.next = page
					case "prev":
						result.prev = page
					case "first":
						result.first = page
					case "last":
						result.last = page
					}
				}
			}
		}
	}
	return result
}

func splitLinkEntries(header string) []string {
	var entries []string
	start := 0
	inAngle := false
	inQuote := false
	for index, char := range header {
		switch char {
		case '<':
			if !inQuote {
				inAngle = true
			}
		case '>':
			if !inQuote {
				inAngle = false
			}
		case '"':
			if !inAngle {
				inQuote = !inQuote
			}
		case ',':
			if !inAngle && !inQuote {
				entries = append(entries, strings.TrimSpace(header[start:index]))
				start = index + 1
			}
		}
	}
	if entry := strings.TrimSpace(header[start:]); entry != "" {
		entries = append(entries, entry)
	}
	return entries
}

func positiveInt(value string) (int, error) {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return 0, errors.New("not a positive integer")
	}
	return parsed, nil
}

// addOptions preserves query values already present in baseURL and adds fields
// tagged with `url`. Embedded option structs are traversed without mutating the
// caller's value. Zero integers, empty strings, false booleans, and nil slices
// are omitted when their tag includes omitempty.
func addOptions(baseURL string, options any) (string, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return baseURL, err
	}
	query := parsed.Query()
	if options == nil {
		parsed.RawQuery = query.Encode()
		return parsed.String(), nil
	}
	if err := addReflectedOptions(reflect.ValueOf(options), query); err != nil {
		return baseURL, err
	}
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func addReflectedOptions(value reflect.Value, query url.Values) error {
	for value.IsValid() && (value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface) {
		if value.IsNil() {
			return nil
		}
		value = value.Elem()
	}
	if !value.IsValid() || value.Kind() != reflect.Struct {
		return nil
	}

	valueType := value.Type()
	for index := 0; index < value.NumField(); index++ {
		fieldType := valueType.Field(index)
		fieldValue := value.Field(index)
		if fieldType.Anonymous {
			if err := addReflectedOptions(fieldValue, query); err != nil {
				return err
			}
			continue
		}
		tag := fieldType.Tag.Get("url")
		if tag == "" || tag == "-" {
			continue
		}
		parts := strings.Split(tag, ",")
		name := parts[0]
		if name == "" {
			name = fieldType.Name
		}
		omitEmpty := false
		for _, option := range parts[1:] {
			if option == "omitempty" {
				omitEmpty = true
			}
		}
		values, empty, err := optionStringValues(fieldValue)
		if err != nil {
			return fmt.Errorf("github: encoding %s: %w", fieldType.Name, err)
		}
		if omitEmpty && empty {
			continue
		}
		query.Del(name)
		for _, optionValue := range values {
			query.Add(name, optionValue)
		}
	}
	return nil
}

func optionStringValues(value reflect.Value) ([]string, bool, error) {
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil, true, nil
		}
		return optionStringValues(value.Elem())
	}
	switch value.Kind() {
	case reflect.String:
		text := value.String()
		return []string{text}, text == "", nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		number := value.Int()
		return []string{strconv.FormatInt(number, 10)}, number == 0, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		number := value.Uint()
		return []string{strconv.FormatUint(number, 10)}, number == 0, nil
	case reflect.Bool:
		flag := value.Bool()
		return []string{strconv.FormatBool(flag)}, !flag, nil
	case reflect.Slice, reflect.Array:
		if value.Len() == 0 {
			return nil, true, nil
		}
		values := make([]string, 0, value.Len())
		for index := 0; index < value.Len(); index++ {
			item, _, err := optionStringValues(value.Index(index))
			if err != nil {
				return nil, false, err
			}
			values = append(values, item...)
		}
		return values, false, nil
	default:
		return nil, false, fmt.Errorf("unsupported kind %s", value.Kind())
	}
}

// ErrorResponse is returned for a non-2xx GitHub API response.
type ErrorResponse struct {
	Response *http.Response
	Message  string
}

func (e *ErrorResponse) Error() string {
	return fmt.Sprintf("github: %s: %s", e.Response.Status, e.Message)
}
