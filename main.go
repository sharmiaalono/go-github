package main

import (
	"fmt"
	"net/url"
	"strconv"
)

// ListOptions specifies the optional parameters to various List methods.
type ListOptions struct {
	Page    int
	PerPage int
}

// NormalizePagination ensures pagination parameters stay within safe bounds.
func NormalizePagination(opts *ListOptions) (int, int) {
	if opts == nil {
		return 1, 30
	}
	page := opts.Page
	if page <= 0 {
		page = 1
	}
	perPage := opts.PerPage
	if perPage <= 0 {
		perPage = 30 // Nilai default yang aman
	} else if perPage > 100 {
		perPage = 100 // Batas maksimal query
	}
	return page, perPage
}

// BuildQuery menyusun parameter kueri URL yang telah divalidasi.
func BuildQuery(opts *ListOptions) string {
	page, perPage := NormalizePagination(opts)
	q := url.Values{}
	q.Set("page", strconv.Itoa(page))
	q.Set("per_page", strconv.Itoa(perPage))
	return q.Encode()
}

func main() {
	opts := &ListOptions{Page: 0, PerPage: 0}
	fmt.Println("Hasil normalisasi query parameter:", BuildQuery(opts))
}
