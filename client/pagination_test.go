package client

import (
	"net/http"
	"strings"
	"testing"
)

func TestAddOptions_OmitsPerPageWhenZero(t *testing.T) {
	opts := ListOptions{Page: 1, PerPage: 0}
	result, err := addOptions("https://api.github.com/repos/foo/bar/issues", &opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(result, "per_page=0") {
		t.Errorf("expected per_page=0 to be omitted, got: %s", result)
	}
	if !strings.Contains(result, "page=1") {
		t.Errorf("expected page=1 to be present, got: %s", result)
	}
}

func TestAddOptions_IncludesPerPageWhenNonZero(t *testing.T) {
	opts := ListOptions{Page: 2, PerPage: 10}
	result, err := addOptions("https://api.github.com/repos/foo/bar/issues", &opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "per_page=10") {
		t.Errorf("expected per_page=10 to be present, got: %s", result)
	}
}

func TestAddOptions_MergesWithExistingQuery(t *testing.T) {
	opts := ListOptions{Page: 1, PerPage: 25}
	result, err := addOptions("https://api.github.com/repos/foo/bar/issues?state=open", &opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "state=open") {
		t.Errorf("expected state=open to be preserved, got: %s", result)
	}
	if !strings.Contains(result, "page=1") {
		t.Errorf("expected page=1 to be present, got: %s", result)
	}
}

func TestParsePagination_NoLinkHeader(t *testing.T) {
	resp := &http.Response{Header: http.Header{}}
	result, err := parsePagination(resp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.NextPage != 0 || result.PrevPage != 0 {
		t.Errorf("expected zero page values, got next=%d prev=%d", result.NextPage, result.PrevPage)
	}
}

func TestParsePagination_WithLinkHeader(t *testing.T) {
	resp := &http.Response{Header: http.Header{}}
	resp.Header.Set("Link", `<https://api.github.com/repos/foo/bar/issues?page=2>; rel="next", <https://api.github.com/repos/foo/bar/issues?page=5>; rel="last"`)
	result, err := parsePagination(resp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.NextPage != 2 {
		t.Errorf("expected NextPage=2, got %d", result.NextPage)
	}
	if result.LastPage != 5 {
		t.Errorf("expected LastPage=5, got %d", result.LastPage)
	}
}

func TestParsePagination_IgnoresPerPageParam(t *testing.T) {
	resp := &http.Response{Header: http.Header{}}
	resp.Header.Set("Link", `<https://api.github.com/repos/foo/bar/issues?page=3&per_page=50>; rel="next"`)
	result, err := parsePagination(resp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.NextPage != 3 {
		t.Errorf("expected NextPage=3, got %d", result.NextPage)
	}
}

func TestParsePagination_MissingPageParam(t *testing.T) {
	resp := &http.Response{Header: http.Header{}}
	resp.Header.Set("Link", `<https://api.github.com/repos/foo/bar/issues?per_page=50>; rel="next"`)
	result, err := parsePagination(resp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.NextPage != 0 {
		t.Errorf("expected NextPage=0 when page param missing, got %d", result.NextPage)
	}
}

func TestParsePagination_FirstAndPrev(t *testing.T) {
	resp := &http.Response{Header: http.Header{}}
	resp.Header.Set("Link", `<https://api.github.com/repos/foo/bar/issues?page=1>; rel="first", <https://api.github.com/repos/foo/bar/issues?page=2>; rel="prev"`)
	result, err := parsePagination(resp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.FirstPage != 1 {
		t.Errorf("expected FirstPage=1, got %d", result.FirstPage)
	}
	if result.PrevPage != 2 {
		t.Errorf("expected PrevPage=2, got %d", result.PrevPage)
	}
}
