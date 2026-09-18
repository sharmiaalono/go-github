// Package github implements GitHub API pagination helpers for ListOptions.
//
// The GitHub REST API treats a missing per_page query parameter as "use the
// server default" (typically 30). Go's zero value for int is also 0, so a
// caller who omits PerPage would otherwise encode per_page=0, which is not a
// valid page size and makes follow-up page requests diverge from the first.
package github

import (
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
)

// DefaultPerPage is the GitHub REST API default page size used when PerPage is
// left at its zero value. Helpers never send per_page=0; they either omit the
// parameter or reuse a per_page value parsed from a Link header.
const DefaultPerPage = 30

// ListOptions specifies the optional parameters to list methods that support
// offset pagination.
type ListOptions struct {
	// Page is the page of results to retrieve.
	Page int `url:"page,omitempty"`

	// PerPage is the number of results to include per page.
	// A zero value means "unspecified": the per_page query parameter is
	// omitted so the API can apply its default.
	PerPage int `url:"per_page,omitempty"`
}

// Response wraps an HTTP response and exposes pagination values parsed from
// the RFC 5988 Link header.
type Response struct {
	*http.Response

	NextPage  int
	PrevPage  int
	FirstPage int
	LastPage  int

	// PerPage is the per_page value extracted from Link URLs when present.
	// It stays 0 when the Link header only carries page parameters.
	PerPage int
}

// newResponse wraps r and parses pagination metadata.
func newResponse(r *http.Response) *Response {
	resp := &Response{Response: r}
	if r != nil {
		parsePagination(resp)
	}
	return resp
}

// AddOptions encodes opts as URL query parameters and merges them into s.
// Zero-valued fields tagged with omitempty are skipped, so PerPage == 0 never
// produces per_page=0. Existing query parameters on s are preserved unless a
// non-zero option overwrites the same key.
func AddOptions(s string, opts any) (string, error) {
	return addOptions(s, opts)
}

func addOptions(s string, opts any) (string, error) {
	if opts == nil {
		return s, nil
	}

	v := reflect.ValueOf(opts)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return s, nil
		}
	}

	u, err := url.Parse(s)
	if err != nil {
		return s, err
	}

	qs, err := valuesFrom(opts)
	if err != nil {
		return s, err
	}

	q := u.Query()
	for key, vals := range qs {
		for _, val := range vals {
			q.Set(key, val)
		}
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// BuildURL applies pagination options to base. When PerPage is 0 the existing
// per_page query value (if any) is left untouched so subsequent page requests
// stay consistent with the first response.
func BuildURL(base string, opts ListOptions) (string, error) {
	return buildURL(base, opts)
}

func buildURL(base string, opts ListOptions) (string, error) {
	u, err := url.Parse(base)
	if err != nil {
		return "", err
	}

	q := u.Query()
	if opts.Page > 0 {
		q.Set("page", strconv.Itoa(opts.Page))
	}

	switch {
	case opts.PerPage > 0:
		q.Set("per_page", strconv.Itoa(opts.PerPage))
	case opts.PerPage < 0:
		return "", fmt.Errorf("ListOptions.PerPage must be >= 0, got %d", opts.PerPage)
	default:
		// PerPage == 0: do not emit per_page=0 and do not strip an existing
		// per_page that came from a previous Link URL or caller.
	}

	u.RawQuery = q.Encode()
	return u.String(), nil
}

// ParsePagination populates page pointers on resp from its Link header.
func ParsePagination(resp *Response) {
	parsePagination(resp)
}

func parsePagination(resp *Response) {
	if resp == nil || resp.Response == nil {
		return
	}

	header := resp.Header.Get("Link")
	if header == "" {
		return
	}

	for _, part := range strings.Split(header, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		href, rel := parseLinkPart(part)
		if href == "" || rel == "" {
			continue
		}

		u, err := url.Parse(href)
		if err != nil {
			continue
		}

		q := u.Query()
		pageStr := q.Get("page")
		if pageStr == "" {
			continue
		}

		page, err := strconv.Atoi(pageStr)
		if err != nil {
			continue
		}

		if perPageStr := q.Get("per_page"); perPageStr != "" {
			if perPage, err := strconv.Atoi(perPageStr); err == nil && perPage > 0 {
				resp.PerPage = perPage
			}
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

func parseLinkPart(part string) (href, rel string) {
	segments := strings.Split(part, ";")
	if len(segments) < 2 {
		return "", ""
	}

	rawHref := strings.TrimSpace(segments[0])
	if !strings.HasPrefix(rawHref, "<") || !strings.HasSuffix(rawHref, ">") {
		return "", ""
	}
	href = strings.TrimSuffix(strings.TrimPrefix(rawHref, "<"), ">")

	for _, segment := range segments[1:] {
		segment = strings.TrimSpace(segment)
		key, val, ok := strings.Cut(segment, "=")
		if !ok || strings.TrimSpace(key) != "rel" {
			continue
		}
		rel = strings.Trim(strings.TrimSpace(val), `"`)
		break
	}
	return href, rel
}

// NextPageOptions returns options for the next page, preserving an explicit
// PerPage from current or falling back to the per_page parsed from Link.
func (r *Response) NextPageOptions(current ListOptions) ListOptions {
	if r == nil || r.NextPage == 0 {
		return ListOptions{}
	}
	return r.pageOptions(r.NextPage, current)
}

// PrevPageOptions returns options for the previous page.
func (r *Response) PrevPageOptions(current ListOptions) ListOptions {
	if r == nil || r.PrevPage == 0 {
		return ListOptions{}
	}
	return r.pageOptions(r.PrevPage, current)
}

// FirstPageOptions returns options for the first page.
func (r *Response) FirstPageOptions(current ListOptions) ListOptions {
	if r == nil || r.FirstPage == 0 {
		return ListOptions{}
	}
	return r.pageOptions(r.FirstPage, current)
}

// LastPageOptions returns options for the last page.
func (r *Response) LastPageOptions(current ListOptions) ListOptions {
	if r == nil || r.LastPage == 0 {
		return ListOptions{}
	}
	return r.pageOptions(r.LastPage, current)
}

func (r *Response) pageOptions(page int, current ListOptions) ListOptions {
	perPage := current.PerPage
	if perPage == 0 {
		perPage = r.PerPage
	}
	return ListOptions{Page: page, PerPage: perPage}
}

func valuesFrom(opts any) (url.Values, error) {
	qs := make(url.Values)
	if err := appendValues(qs, reflect.ValueOf(opts)); err != nil {
		return nil, err
	}
	return qs, nil
}

func appendValues(qs url.Values, v reflect.Value) error {
	if !v.IsValid() {
		return nil
	}

	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return fmt.Errorf("addOptions: opts must be a struct or pointer to struct, got %s", v.Kind())
	}

	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		fv := v.Field(i)
		if field.Anonymous {
			if err := appendValues(qs, fv); err != nil {
				return err
			}
			continue
		}

		tag := field.Tag.Get("url")
		if tag == "" || tag == "-" {
			continue
		}

		name, opts, _ := strings.Cut(tag, ",")
		if name == "" {
			name = field.Name
		}
		omitempty := strings.Contains(opts, "omitempty")

		if fv.Kind() == reflect.Ptr {
			if fv.IsNil() {
				continue
			}
			fv = fv.Elem()
		}

		if omitempty && isEmptyValue(fv) {
			continue
		}

		qs.Set(name, valueString(fv))
	}
	return nil
}

func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	default:
		return false
	}
}

func valueString(v reflect.Value) string {
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return strconv.FormatUint(v.Uint(), 10)
	case reflect.Bool:
		return strconv.FormatBool(v.Bool())
	default:
		return fmt.Sprint(v.Interface())
	}
}
