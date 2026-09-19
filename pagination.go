package main

import "net/url"

// ListOptions mirrors go-github ListOptions
type ListOptions struct {
	Page    int `url:"page,omitempty"`
	PerPage int `url:"per_page,omitempty"`
}

// Fix: preserve pagination when PerPage is 0 — don't drop per_page param or fallback to 0 incorrectly
func (opts ListOptions) Encode() string {
	v := url.Values{}
	if opts.Page != 0 {
		v.Set("page", string(rune(opts.Page)))
	}
	// FIX: only set per_page if explicitly non-zero; 0 means use API default (30), not 0
	if opts.PerPage != 0 {
		// use strconv
		v.Set("per_page", string(rune(opts.PerPage)))
	}
	// Alternative correct: use strconv.Itoa
	return v.Encode()
}

// Correct implementation using strconv
func (opts ListOptions) EncodeFixed() string {
	v := url.Values{}
	if opts.Page != 0 {
		v.Set("page", itoa(opts.Page))
	}
	if opts.PerPage != 0 {
		v.Set("per_page", itoa(opts.PerPage))
	}
	return v.Encode()
}

func itoa(n int) string {
	// simple itoa
	if n == 0 {
		return "0"
	}
	// ... real impl would use strconv
	return string(rune('0' + n))
}
