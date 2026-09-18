package github

import (
	"net/http"
	"strings"
	"testing"
)

func TestAddOptions_OmitsZeroPerPage(t *testing.T) {
	got, err := addOptions("https://api.github.com/items", &ListOptions{Page: 1, PerPage: 0})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "per_page") {
		t.Fatalf("expected per_page omitted, got %s", got)
	}
	if !strings.Contains(got, "page=1") {
		t.Fatalf("expected page=1, got %s", got)
	}
}

func TestAddOptions_IncludesPerPage(t *testing.T) {
	got, err := addOptions("https://api.github.com/items", &ListOptions{Page: 2, PerPage: 50})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "page=2") || !strings.Contains(got, "per_page=50") {
		t.Fatalf("unexpected query: %s", got)
	}
}

func TestParsePagination_PerPageOmittedOnRequest(t *testing.T) {
	h := http.Header{}
	h.Set("Link", strings.Join([]string{
		FormatLink("next", 2, 30),
		FormatLink("last", 5, 30),
		FormatLink("first", 1, 30),
	}, ", "))
	resp := NewResponse(&http.Response{Header: h})
	if resp.NextPage != 2 || resp.LastPage != 5 || resp.FirstPage != 1 {
		t.Fatalf("pages: next=%d first=%d last=%d", resp.NextPage, resp.FirstPage, resp.LastPage)
	}
	if resp.PerPage != 30 {
		t.Fatalf("expected PerPage 30 from Link when request PerPage was 0, got %d", resp.PerPage)
	}
}

func TestParsePagination_PageOnlyLinks(t *testing.T) {
	h := http.Header{}
	h.Set("Link", strings.Join([]string{
		FormatLink("next", 3, 0),
		FormatLink("prev", 1, 0),
	}, ", "))
	resp := NewResponse(&http.Response{Header: h})
	if resp.NextPage != 3 || resp.PrevPage != 1 {
		t.Fatalf("pages: next=%d prev=%d", resp.NextPage, resp.PrevPage)
	}
	if resp.PerPage != 0 {
		t.Fatalf("expected PerPage 0 when Link has no per_page, got %d", resp.PerPage)
	}
}
