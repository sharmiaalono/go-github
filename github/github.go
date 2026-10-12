// Package github provides helpers for GitHub API list pagination.
//
// This package focuses on ListOptions / Response behavior when PerPage is 0:
// the per_page query parameter is omitted (GitHub applies its default page
// size), while Link-header page helpers still resolve correctly.
package github

import (
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
)

// ListOptions specifies optional pagination parameters for list endpoints.
//
// When PerPage is 0, the per_page query parameter is omitted from the request
// (via the url:",omitempty" tag). The GitHub API then applies its standard
// default page size (typically 30). Page numbers from subsequent Link headers
// remain authoritative for NextPage / PrevPage / FirstPage / LastPage.
type ListOptions struct {
	// Page is the page number to request. Zero omits the page parameter.
	Page int `url:"page,omitempty"`

	// PerPage is the page size. Zero omits per_page so the server default applies.
	PerPage int `url:"per_page,omitempty"`
}

// Response wraps an HTTP response and exposes pagination cursors parsed from
// the RFC 5988 Link header. A zero page value means that relation was absent.
type Response struct {
	*http.Response

	NextPage  int
	PrevPage  int
	FirstPage int
	LastPage  int

	// Rate may be filled by callers that parse X-RateLimit-* headers.
	Rate Rate
}

// Rate represents the GitHub API rate limit counters when present.
type Rate struct {
	Limit     int
	Remaining int
	Reset     int64
}

// NewResponse builds a Response and parses Link pagination relations.
func NewResponse(r *http.Response) *Response {
	resp := &Response{Response: r}
	if r != nil {
		resp.parsePagination()
	}
	return resp
}

// parsePagination extracts page numbers from the Link header.
// It reads only the page query parameter from each rel URL, so it works
// whether or not per_page appears in the Link target (including when the
// original request omitted per_page because ListOptions.PerPage was 0).
func (r *Response) parsePagination() {
	if r.Response == nil {
		return
	}
	links := r.Header.Values("Link")
	if len(links) == 0 {
		// Some transports fold multi-value headers; also check Get.
		if v := r.Header.Get("Link"); v != "" {
			links = []string{v}
		}
	}
	if len(links) == 0 {
		return
	}

	for _, link := range strings.Split(strings.Join(links, ","), ",") {
		link = strings.TrimSpace(link)
		parts := strings.Split(link, ";")
		if len(parts) < 2 {
			continue
		}
		uStr := strings.Trim(parts[0], " <>")
		var rel string
		for _, p := range parts[1:] {
			p = strings.TrimSpace(p)
			if strings.HasPrefix(strings.ToLower(p), "rel=") {
				rel = strings.Trim(p[4:], `"`)
				break
			}
		}
		if rel == "" {
			continue
		}
		u, err := url.Parse(uStr)
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
			r.NextPage = page
		case "prev", "previous":
			r.PrevPage = page
		case "first":
			r.FirstPage = page
		case "last":
			r.LastPage = page
		}
	}
}

// addOptions encodes opts (struct with `url` tags) as query parameters on s.
// Fields tagged omitempty are skipped when they hold their zero value, so
// ListOptions{PerPage: 0} does not emit per_page.
func addOptions(s string, opts any) (string, error) {
	if opts == nil {
		return s, nil
	}
	v := reflect.ValueOf(opts)
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return s, nil
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return s, fmt.Errorf("addOptions: opts must be a struct, got %s", v.Kind())
	}

	u, err := url.Parse(s)
	if err != nil {
		return s, err
	}
	q := u.Query()
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("url")
		if tag == "" || tag == "-" {
			continue
		}
		name, optsTag, _ := strings.Cut(tag, ",")
		if name == "" {
			continue
		}
		omitempty := strings.Contains(optsTag, "omitempty")
		fv := v.Field(i)
		if omitempty && isEmptyValue(fv) {
			continue
		}
		q.Set(name, valueToString(fv))
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
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
	case reflect.Interface, reflect.Pointer:
		return v.IsNil()
	}
	return false
}

func valueToString(v reflect.Value) string {
	switch v.Kind() {
	case reflect.String:
		return v.String()
	case reflect.Bool:
		return strconv.FormatBool(v.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return strconv.FormatUint(v.Uint(), 10)
	default:
		return fmt.Sprint(v.Interface())
	}
}

// WithPage returns a copy of opts with Page set, preserving PerPage.
// Useful when following Response.NextPage without dropping an explicit page size.
func (o ListOptions) WithPage(page int) ListOptions {
	o.Page = page
	return o
}
