// README.md
# Pagination Defaults

The client library normalizes pagination options to provide consistent and safe defaults across all endpoints.

- **PerPage**: If omitted (zero value) or explicitly set to `0`, the library will use `DefaultPerPage` (currently `30`).
- **Page**: If omitted or set to `0`, the library defaults to the first page (`1`).
- Values exceeding the limits are capped (`PerPage` is capped at `MaxPerPage`).

This behavior is implemented by `normalizeListOptions` and is applied automatically by `BuildListURL` and the `SetDefaults` helper on `ListOptions`.

---

// main.go
package main

import (
    "fmt"
    "net/url"
    "strconv"
)

type ListOptions struct {
    // PerPage specifies how many items should be returned per page.
    // If it is zero or omitted, the default value `DefaultPerPage` (30) is used.
    // The value is capped at `MaxPerPage` (100).
    PerPage int
    // Page specifies which page of results to retrieve.
    // If it is zero or omitted, the request defaults to the first page (1).
    Page int
}

// DefaultPerPage is the safe default number of items per page.
const DefaultPerPage = 30

// MaxPerPage is the enforced upper bound for items per page.
const MaxPerPage = 100

// normalizeListOptions sets safe defaults and caps values.
func normalizeListOptions(opt *ListOptions) {
    if opt == nil {
        return
    }
    // Ensure a sensible PerPage value.
    if opt.PerPage <= 0 {
        opt.PerPage = DefaultPerPage
    }
    if opt.PerPage > MaxPerPage {
        opt.PerPage = MaxPerPage
    }
    // Ensure a sensible Page value.
    if opt.Page <= 0 {
        opt.Page = 1
    }
}

// SetDefaults ensures PerPage and Page have safe defaults.
func (opt *ListOptions) SetDefaults() {
    normalizeListOptions(opt)
}

// BuildListURL constructs a request URL with normalized list options.
// It is a generic helper that endpoint implementations can use before
// sending a request, guaranteeing that per_page is always non‑zero and
// within the allowed bounds.
func BuildListURL(base string, opt *ListOptions) (string, error) {
    // If no options are provided, use a fully defaulted ListOptions.
    if opt == nil {
        defaultOpt := ListOptions{}
        normalizeListOptions(&defaultOpt)
        opt = &defaultOpt
    } else {
        // Ensure the provided options are normalized first.
        normalizeListOptions(opt)
    }

    u, err := url.Parse(base)
    if err != nil {
        return "", err
    }
    q := u.Query()
    // After normalization, PerPage is guaranteed to be >0, so we always include it.
    q.Set("per_page", strconv.Itoa(opt.PerPage))
    q.Set("page", strconv.Itoa(opt.Page))
    u.RawQuery = q.Encode()
    return u.String(), nil
}

func main() {
    // Example usage of ListOptions with safe defaults.
    opts := ListOptions{}
    // Directly using SetDefaults (optional – BuildListURL also normalizes).
    opts.SetDefaults()
    fmt.Printf("PerPage: %d, Page: %d\n", opts.PerPage, opts.Page)

    // Simulate an endpoint building its request URL.
    url, err := BuildListURL("https://api.example.com/items", &opts)
    if err != nil {
        panic(err)
    }
    fmt.Println("Request URL:", url)
    fmt.Println("Hello, Bounty Hunter!")
}

// normalize_test.go
package main

import "testing"

func TestNormalizeListOptions_DefaultsReplaced(t *testing.T) {
    opt := &ListOptions{PerPage: 0, Page: 0}
    normalizeListOptions(opt)
    if opt.PerPage != DefaultPerPage {
        t.Fatalf("expected PerPage to be default %d, got %d", DefaultPerPage, opt.PerPage)
    }
    if opt.Page != 1 {
        t.Fatalf("expected Page to be default 1, got %d", opt.Page)
    }
}

func TestNormalizeListOptions_NonZeroPreserved(t *testing.T) {
    opt := &ListOptions{PerPage: 50, Page: 5}
    normalizeListOptions(opt)
    if opt.PerPage != 50 {
        t.Fatalf("expected PerPage to remain 50, got %d", opt.PerPage)
    }
    if opt.Page != 5 {
        t.Fatalf("expected Page to remain 5, got %d", opt.Page)
    }
}

// endpoint_test.go
package main

import (
    "net/url"
    "testing"
)

func TestListRepositories_DefaultPerPageAndPage(t *testing.T) {
    // Simulate calling ListRepositories with PerPage = 0 and a specific page.
    opt := &ListOptions{PerPage: 0, Page: 3}
    requestURL, err := BuildListURL("https://api.example.com/repos", opt)
    if err != nil {
        t.Fatalf("BuildListURL returned error: %v", err)
    }

    u, err := url.Parse(requestURL)
    if err != nil {
        t.Fatalf("failed to parse generated URL: %v", err)
    }
    q := u.Query()

    // Expect the default per_page value.
    if got := q.Get("per_page"); got != "30" { // DefaultPerPage = 30
        t.Fatalf("expected per_page to be default '30', got '%s'", got)
    }
    // Expect the page to be whatever was supplied (3).
    if got := q.Get("page"); got != "3" {
        t.Fatalf("expected page to be '3', got '%s'", got)
    }
}

func TestListIssues_DefaultPerPageAndPage(t *testing.T) {
    // Simulate calling ListIssues with PerPage = 0 and default page.
    opt := &ListOptions{PerPage: 0, Page: 0}
    requestURL, err := BuildListURL("https://api.example.com/issues", opt)
    if err != nil {
        t.Fatalf("BuildListURL returned error: %v", err)
    }

    u, err := url.Parse(requestURL)
    if err != nil {
        t.Fatalf("failed to parse generated URL: %v", err)
    }
    q := u.Query()

    // Expect the default per_page value.
    if got := q.Get("per_page"); got != "30" {
        t.Fatalf("expected per_page to be default '30', got '%s'", got)
    }
    // Expect the default page value (1) after normalization.
    if got := q.Get("page"); got != "1" {
        t.Fatalf("expected page to be default '1', got '%s'", got)
    }
}
