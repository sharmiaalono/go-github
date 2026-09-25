package main

import (
	"fmt"
	"net/url"
	"strconv"
)

// DefaultPerPage is the safe default number of items per page when PerPage is 0 or omitted.
const DefaultPerPage = 30

// MaxPerPage is the enforced upper bound for items per page.
const MaxPerPage = 100

// ListOptions specifies the optional parameters to list methods.
type ListOptions struct {
	// Page specifies which page of results to retrieve (defaults to 1).
	Page int
	// PerPage specifies how many items should be returned per page (defaults to 30 when 0).
	PerPage int
}

// normalizeListOptions sets safe defaults and caps values when PerPage is 0 or unassigned.
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
// Guarantees that per_page is always consistent and non-zero.
func BuildListURL(base string, opt *ListOptions) (string, error) {
	if opt == nil {
		defaultOpt := ListOptions{}
		normalizeListOptions(&defaultOpt)
		opt = &defaultOpt
	} else {
		normalizeListOptions(opt)
	}

	u, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("per_page", strconv.Itoa(opt.PerPage))
	q.Set("page", strconv.Itoa(opt.Page))
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func main() {
	// Example usage of ListOptions with safe defaults.
	opts := ListOptions{
		Page:    1,
		PerPage: 0, // Zero value relies on default 30
	}
	opts.SetDefaults()
	fmt.Printf("Normalized ListOptions: Page=%d, PerPage=%d\n", opts.Page, opts.PerPage)

	requestURL, err := BuildListURL("https://api.github.com/repos/owner/repo/issues", &opts)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Println("Generated API URL:", requestURL)
	fmt.Println("Hello, Bounty Hunter!")
}
