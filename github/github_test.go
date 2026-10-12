package github

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestAddOptions_PerPageZeroOmitsParam(t *testing.T) {
	got, err := addOptions("https://api.github.com/user/repos", ListOptions{Page: 1, PerPage: 0})
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(got)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	if q.Get("page") != "1" {
		t.Fatalf("page=%q, want 1", q.Get("page"))
	}
	if _, ok := q["per_page"]; ok {
		t.Fatalf("per_page should be omitted when PerPage=0, got %q", q.Get("per_page"))
	}
}

func TestAddOptions_PerPageNonZeroIncluded(t *testing.T) {
	got, err := addOptions("https://api.github.com/user/repos", ListOptions{Page: 2, PerPage: 50})
	if err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(got)
	q := u.Query()
	if q.Get("page") != "2" || q.Get("per_page") != "50" {
		t.Fatalf("got query %v", q)
	}
}

func TestAddOptions_NilAndEmpty(t *testing.T) {
	base := "https://api.github.com/repos/o/r/issues"
	got, err := addOptions(base, nil)
	if err != nil || got != base {
		t.Fatalf("nil opts: got %q err=%v", got, err)
	}
	got, err = addOptions(base, ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "page=") || strings.Contains(got, "per_page=") {
		t.Fatalf("zero ListOptions should omit both params, got %q", got)
	}
}

func TestParsePagination_WithoutPerPageInLink(t *testing.T) {
	// Simulates GitHub Link header after a request that omitted per_page.
	link := `<https://api.github.com/user/repos?page=2>; rel="next", ` +
		`<https://api.github.com/user/repos?page=1>; rel="first", ` +
		`<https://api.github.com/user/repos?page=4>; rel="last"`
	httpResp := &http.Response{Header: make(http.Header)}
	httpResp.Header.Set("Link", link)
	resp := NewResponse(httpResp)
	if resp.NextPage != 2 || resp.FirstPage != 1 || resp.LastPage != 4 {
		t.Fatalf("pages next=%d first=%d last=%d", resp.NextPage, resp.FirstPage, resp.LastPage)
	}
	if resp.PrevPage != 0 {
		t.Fatalf("prev should be 0, got %d", resp.PrevPage)
	}
}

func TestParsePagination_WithPerPageInLink(t *testing.T) {
	link := `<https://api.github.com/user/repos?page=3&per_page=50>; rel="next", ` +
		`<https://api.github.com/user/repos?page=1&per_page=50>; rel="prev", ` +
		`<https://api.github.com/user/repos?page=1&per_page=50>; rel="first", ` +
		`<https://api.github.com/user/repos?page=10&per_page=50>; rel="last"`
	httpResp := &http.Response{Header: make(http.Header)}
	httpResp.Header.Set("Link", link)
	resp := NewResponse(httpResp)
	if resp.NextPage != 3 || resp.PrevPage != 1 || resp.FirstPage != 1 || resp.LastPage != 10 {
		t.Fatalf("got next=%d prev=%d first=%d last=%d", resp.NextPage, resp.PrevPage, resp.FirstPage, resp.LastPage)
	}
}

func TestWithPage_PreservesPerPage(t *testing.T) {
	opts := ListOptions{Page: 1, PerPage: 50}.WithPage(2)
	if opts.Page != 2 || opts.PerPage != 50 {
		t.Fatalf("got %+v", opts)
	}
	// Following next page after a PerPage=0 initial request still omits per_page.
	opts0 := ListOptions{Page: 1, PerPage: 0}.WithPage(2)
	got, err := addOptions("https://api.github.com/user/repos", opts0)
	if err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(got)
	if u.Query().Get("page") != "2" {
		t.Fatalf("page=%q", u.Query().Get("page"))
	}
	if _, ok := u.Query()["per_page"]; ok {
		t.Fatal("per_page must stay omitted when PerPage remains 0")
	}
}

func TestParsePagination_EmptyLink(t *testing.T) {
	resp := NewResponse(&http.Response{Header: make(http.Header)})
	if resp.NextPage != 0 || resp.PrevPage != 0 {
		t.Fatalf("expected zeros, got %+v", resp)
	}
}
