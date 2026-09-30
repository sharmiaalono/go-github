package main

import (
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"strings"
)

// ListOptions specifies the optional parameters to various List methods.
// The zero value means no pagination (use server defaults).
type ListOptions struct {
	// For paginated result sets. Page of results to retrieve.
	Page int `url:"page,omitempty"`

	// For paginated result sets. Number of results per page.
	// If 0, the server default (30) is used.
	PerPage int `url:"per_page,omitempty"`
}

// Pagination tracks pagination state extracted from the Link header.
type Pagination struct {
	NextPage  int
	PrevPage  int
	FirstPage int
	LastPage  int
}

// Response is a GitHub API response. It embeds http.Response and adds
// pagination helpers that parse the Link header.
type Response struct {
	// These fields are set from the Link header
	FirstPage int
	PrevPage  int
	NextPage  int
	LastPage  int
}

// parsePagination extracts page numbers from the Link header value.
// The Link header uses the format: <url>; rel="next", <url>; rel="last"
// Page numbers are extracted from the URL query parameters.
func parsePagination(raw string) Pagination {
	p := Pagination{}
	if raw == "" {
		return p
	}
	links := strings.Split(raw, ",")
	for _, link := range links {
		link = strings.TrimSpace(link)
		// Format: <url>; rel="type"
		parts := strings.SplitN(link, ";", 2)
		if len(parts) < 2 {
			continue
		}
		urlPart := strings.TrimSpace(parts[0])
		urlPart = strings.Trim(urlPart, "<>")
		relPart := strings.TrimSpace(parts[1])

		// Extract page number from URL
		page := extractPageFromURL(urlPart)
		if page == 0 {
			continue
		}

		// Extract rel type
		rel := extractRel(relPart)
		switch rel {
		case "next":
			p.NextPage = page
		case "prev":
			p.PrevPage = page
		case "first":
			p.FirstPage = page
		case "last":
			p.LastPage = page
		}
	}
	return p
}

// extractPageFromURL pulls the "page" query parameter from a URL string.
func extractPageFromURL(rawURL string) int {
	u, err := url.Parse(rawURL)
	if err != nil {
		return 0
	}
	pageStr := u.Query().Get("page")
	if pageStr == "" {
		return 0
	}
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		return 0
	}
	return page
}

// extractRel pulls the rel type from the Link header part.
func extractRel(relPart string) string {
	// Format: rel="next"
	for _, kv := range strings.Split(relPart, ";") {
		kv = strings.TrimSpace(kv)
		if strings.HasPrefix(kv, "rel=") {
			val := strings.TrimPrefix(kv, "rel=")
			val = strings.Trim(val, "\"")
			return val
		}
	}
	return ""
}

// addOptions adds the parameters in opts as URL query parameters.
// opts must be a struct whose fields contain url tags.
func addOptions(s string, opts interface{}) string {
	if opts == nil {
		return s
	}
	v := reflect.ValueOf(opts)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return s
	}
	u, err := url.Parse(s)
	if err != nil {
		return s
	}
	q := u.Query()
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("url")
		if tag == "" || tag == "-" {
			continue
		}
		// Parse tag: "page,omitempty" -> key="page", omitempty=true
		parts := strings.Split(tag, ",")
		key := parts[0]
		omitempty := len(parts) > 1 && parts[1] == "omitempty"

		fieldVal := v.Field(i)
		switch fieldVal.Kind() {
		case reflect.Int:
			val := int(fieldVal.Int())
			if val == 0 && omitempty {
				continue
			}
			q.Set(key, strconv.Itoa(val))
		}
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// ensurePerPageDefaults returns perPage if set, or the GitHub API default (30).
func ensurePerPageDefaults(perPage int) int {
	if perPage <= 0 {
		return 30
	}
	return perPage
}

func main() {
	// Demo: ListOptions with PerPage=0 should default to 30
	opts := ListOptions{Page: 2, PerPage: 0}
	_ = ensurePerPageDefaults(opts.PerPage) // returns 30

	// Demo: addOptions encodes the struct
	result := addOptions("https://api.github.com/repos/test/issues", opts)
	fmt.Printf("URL with options: %s\\n", result)

	// Demo: parsePagination extracts page numbers from Link header
	link := `<https://api.github.com/repositories/1/issues?page=2>; rel="prev", <https://api.github.com/repositories/1/issues?page=4>; rel="next", <https://api.github.com/repositories/1/issues?page=1>; rel="first", <https://api.github.com/repositories/1/issues?page=10>; rel="last"`
	p := parsePagination(link)
	fmt.Printf("Pagination: first=%d prev=%d next=%d last=%d\\n", p.FirstPage, p.PrevPage, p.NextPage, p.LastPage)

	// Verify: PerPage=0 doesn't break pagination parsing
	fmt.Printf("PerPage defaults to: %d\\n", ensurePerPageDefaults(0))
}
