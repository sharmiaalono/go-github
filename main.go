package main

import (
	"fmt"
	"net/url"
	"strconv"
)

// DefaultPerPage defines the default items per page when PerPage is 0
const DefaultPerPage = 30

// ListOptions specifies the optional parameters to list methods
type ListOptions struct {
	Page    int
	PerPage int
}

// AddOptions constructs URL parameters from ListOptions.
// Fixes inconsistency: when PerPage is 0, it safely defaults to DefaultPerPage (30).
func AddOptions(baseURL string, opts *ListOptions) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return baseURL, err
	}

	if opts == nil {
		return u.String(), nil
	}

	q := u.Query()

	if opts.Page > 0 {
		q.Set("page", strconv.Itoa(opts.Page))
	}

	// Handle ListOptions.PerPage == 0 consistently
	perPage := opts.PerPage
	if perPage <= 0 {
		perPage = DefaultPerPage
	}
	q.Set("per_page", strconv.Itoa(perPage))

	u.RawQuery = q.Encode()
	return u.String(), nil
}

func main() {
	// Test case where PerPage is set to 0
	opts := &ListOptions{
		Page:    1,
		PerPage: 0,
	}

	endpoint := "https://api.github.com/repos/owner/repo/issues"
	resultURL, err := AddOptions(endpoint, opts)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println("Original PerPage:", opts.PerPage)
	fmt.Println("Generated URL with fix:", resultURL)
}
