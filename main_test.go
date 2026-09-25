package main

import (
	"net/url"
	"testing"
)

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

func TestListRepositories_DefaultPerPageAndPage(t *testing.T) {
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

	if got := q.Get("per_page"); got != "30" {
		t.Fatalf("expected per_page to be default '30', got '%s'", got)
	}
	if got := q.Get("page"); got != "3" {
		t.Fatalf("expected page to be '3', got '%s'", got)
	}
}

func TestListIssues_DefaultPerPageAndPage(t *testing.T) {
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

	if got := q.Get("per_page"); got != "30" {
		t.Fatalf("expected per_page to be default '30', got '%s'", got)
	}
	if got := q.Get("page"); got != "1" {
		t.Fatalf("expected page to be default '1', got '%s'", got)
	}
}
