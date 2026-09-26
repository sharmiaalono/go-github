package client

import (
	"net/http"
	"testing"
)

func TestAddOptions(t *testing.T) {
	t.Run("omits per_page when zero", func(t *testing.T) {
		url, err := addOptions("https://api.example.com/items", &ListOptions{Page: 2, PerPage: 0})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		exp := "https://api.example.com/items?page=2"
		if url != exp {
			t.Errorf("expected %q, got %q", exp, url)
		}
	})

	t.Run("includes per_page when non-zero", func(t *testing.T) {
		url, err := addOptions("https://api.example.com/items", &ListOptions{Page: 1, PerPage: 30})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		exp := "https://api.example.com/items?page=1&per_page=30"
		if url != exp {
			t.Errorf("expected %q, got %q", exp, url)
		}
	})

	t.Run("preserves existing query parameters", func(t *testing.T) {
		url, err := addOptions("https://api.example.com/items?sort=desc", &ListOptions{Page: 1, PerPage: 0})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		exp := "https://api.example.com/items?page=1&sort=desc"
		if url != exp {
			t.Errorf("expected %q, got %q", exp, url)
		}
	})
}

func TestParsePagination(t *testing.T) {
	t.Run("parses link header correctly", func(t *testing.T) {
		resp := &http.Response{
			Header: make(http.Header),
		}
		resp.Header.Set("Link", `<https://api.example.com/items?page=2&per_page=10>; rel="next", <https://api.example.com/items?page=5&per_page=10>; rel="last"`)

		pResp, err := parsePagination(resp)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if pResp.NextPage != 2 {
			t.Errorf("expected NextPage 2, got %d", pResp.NextPage)
		}
		if pResp.LastPage != 5 {
			t.Errorf("expected LastPage 5, got %d", pResp.LastPage)
		}
		if pResp.PrevPage != 0 {
			t.Errorf("expected PrevPage 0, got %d", pResp.PrevPage)
		}
	})

	t.Run("handles missing link header", func(t *testing.T) {
		resp := &http.Response{
			Header: make(http.Header),
		}

		pResp, err := parsePagination(resp)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if pResp.NextPage != 0 || pResp.PrevPage != 0 || pResp.FirstPage != 0 || pResp.LastPage != 0 {
			t.Errorf("expected all page fields to be zero, got %+v", pResp)
		}
	})
}
