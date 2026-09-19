package main

import (
	"net/url"
	"strconv"
)

// ListOptionsFixed is the corrected version
type ListOptionsFixed struct {
	Page    int `url:"page,omitempty"`
	PerPage int `url:"per_page,omitempty"`
}

func (opts ListOptionsFixed) URLParams() url.Values {
	v := url.Values{}
	if opts.Page != 0 {
		v.Set("page", strconv.Itoa(opts.Page))
	}
	// FIX: Preserve PerPage when 0 means default, don't send per_page=0 which breaks pagination
	if opts.PerPage != 0 {
		v.Set("per_page", strconv.Itoa(opts.PerPage))
	}
	// When PerPage==0, omit param so API uses default 30 and pagination Links preserve correct per_page
	return v
}
