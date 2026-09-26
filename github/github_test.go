package github

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestAddOptionsOmitsZeroPerPage(t *testing.T) {
	got, err := addOptions("https://api.github.com/repos/o/r/issues", ListOptions{Page: 1, PerPage: 0})
	if err != nil {
		t.Fatalf("addOptions error: %v", err)
	}
	u, err := url.Parse(got)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if got := u.Query().Get("page"); got != "1" {
		t.Errorf("page = %q, want 1", got)
	}
	if got := u.Query().Get("per_page"); got != "" {
		t.Errorf("per_page = %q, want omitted for PerPage 0", got)
	}
}

func TestAddOptionsKeepsExplicitPerPage(t *testing.T) {
	got, err := addOptions("https://api.github.com/repos/o/r/issues", ListOptions{Page: 2, PerPage: 50})
	if err != nil {
		t.Fatalf("addOptions error: %v", err)
	}
	u, err := url.Parse(got)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if got := u.Query().Get("page"); got != "2" {
		t.Errorf("page = %q, want 2", got)
	}
	if got := u.Query().Get("per_page"); got != "50" {
		t.Errorf("per_page = %q, want 50", got)
	}
}

func TestParsePaginationWithoutPerPage(t *testing.T) {
	// Link header as returned when per_page was omitted (server default).
	link := `<https://api.github.com/repos/o/r/issues?page=2>; rel="next", ` +
		`<https://api.github.com/repos/o/r/issues?page=5>; rel="last"`
	next, prev, first, last := parsePagination(link)
	if next != 2 || prev != 0 || first != 0 || last != 5 {
		t.Errorf("got next=%d prev=%d first=%d last=%d, want 2/0/0/5", next, prev, first, last)
	}
}

func TestParsePaginationWithExplicitPerPage(t *testing.T) {
	// Same page numbers must resolve when per_page is explicit.
	link := `<https://api.github.com/repos/o/r/issues?page=2&per_page=50>; rel="next", ` +
		`<https://api.github.com/repos/o/r/issues?page=1&per_page=50>; rel="prev", ` +
		`<https://api.github.com/repos/o/r/issues?page=1&per_page=50>; rel="first", ` +
		`<https://api.github.com/repos/o/r/issues?page=5&per_page=50>; rel="last"`
	next, prev, first, last := parsePagination(link)
	if next != 2 || prev != 1 || first != 1 || last != 5 {
		t.Errorf("got next=%d prev=%d first=%d last=%d, want 2/1/1/5", next, prev, first, last)
	}
}

func TestPopulatePageValuesIgnoresMalformedRelations(t *testing.T) {
	resp := &http.Response{Header: http.Header{}}
	resp.Header.Set("Link", `<not-a-url>; rel="next", <https://api.github.com/x?page=abc>; rel="last"`)
	var r Response
	r.PopulatePageValues(resp)
	if r.NextPage != 0 || r.LastPage != 0 {
		t.Errorf("malformed relations must yield zeros, got %+v", r)
	}
}

func TestMultiPageFlowKeepsPageSizeContext(t *testing.T) {
	// Simulates GitHub list pagination: first request omits per_page
	// (PerPage 0), Link headers carry explicit per_page, and the client
	// follows NextPage reusing the same ListOptions.
	var seen []string
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.URL.RawQuery)
		page := r.URL.Query().Get("page")
		if page == "" || page == "1" {
			w.Header().Set("Link", fmt.Sprintf(`<%s/issues?page=2&per_page=30>; rel="next", <%s/issues?page=3&per_page=30>; rel="last"`, srv.URL, srv.URL))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	opts := ListOptions{Page: 1, PerPage: 0}
	pages := 0
	for i := 0; i < 3; i++ {
		endpoint, err := addOptions(srv.URL+"/issues", opts)
		if err != nil {
			t.Fatal(err)
		}
		httpResp, err := http.Get(endpoint) //nolint:gosec,noctx
		if err != nil {
			t.Fatal(err)
		}
		var result Response
		result.PopulatePageValues(httpResp)
		httpResp.Body.Close()
		pages++
		if result.NextPage == 0 {
			break
		}
		// Reuse options across pages, advancing only the page number.
		opts.Page = result.NextPage
	}
	if pages != 2 {
		t.Fatalf("expected 2 pages walked, got %d", pages)
	}
	if !strings.Contains(seen[0], "page=1") || strings.Contains(seen[0], "per_page") {
		t.Errorf("first request must carry page without per_page, got %q", seen[0])
	}
	if !strings.Contains(seen[1], "page=2") {
		t.Errorf("follow-up must carry next page, got %q", seen[1])
	}
}
