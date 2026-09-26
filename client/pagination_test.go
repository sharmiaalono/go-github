package client

import (
	"net/http"
	"testing"
)

func TestAddOptions(tests *testing.T) {
	t.Run("omits per_page when zero", func(t *testing.T) {
		u, err := addOptions("https://api.example.com/items", &ListOptions{Page: 2, PerPage: 0})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := "https://api.example.com/items?page=2"
		if u != expected {
			t.Errorf("expected %q, got %q", expected, u)
		}
	})

	t.Run("includes per_page when non-zero", func(t *testing.T) {
		u, err := addOptions("https://api.example.com/items", &ListOptions{Page: 1, PerPage: 30})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := "https://api.example.com/items?page=1&per_page=30"
		if u != expected {
			t.Errorf("expected %q, got %q", expected, u)
		}
	})

	t.Run("preserves existing query params", func(t *testing.T) {
		u, err := addOptions("https://api.example.com/items?sort=desc", &ListOptions{Page: 1, PerPage: 0})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := "https://api.example.com/items?sort=desc&page=1"
		if u != expected {
			t.Errorf("expected %q, got %q", expected, u)
		}
	})
}

func TestParsePagination(t *testing.T) {
	t.Run("parses link header correctly", func(t *testing.T) {
		resp := &http.Response{
			Header: make(http.Header),
		}
		resp.Header.Set("Link", `<https://api.example.com/items?page=2&per_page=30>; rel="next", <https://api.example.com/items?page=5&per_page=30>; rel="last"`)

		p, err := parsePagination(resp)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if p.NextPage != 2 {
			t.Errorf("expected NextPage 2, got %d", p.NextPage)
		}
		if p.LastPage != 5 {
			t.Errorf("expected LastPage 5, got %d", p.LastPage)
		}
		if p.PrevPage != 0 {
			t.Errorf("expected PrevPage 0, got %d", p.PrevPage)
		}
	})

	t.Run("handles missing link header", func(t *testing.T) {
		resp := &http.Response{
			Header: make(http.Header),
		}

		p, err := parsePagination(resp)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if p.NextPage != 0 {
			t.Errorf("expected NextPage 0, got %d", p.NextPage)
		}
	})
}
