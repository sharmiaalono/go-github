package github

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestAddOptions_PerPageZeroOmitsPerPage(t *testing.T) {
	got, err := addOptions("repos/o/r/issues", &ListOptions{Page: 1, PerPage: 0})
	if err != nil {
		t.Fatalf("addOptions returned error: %v", err)
	}
	u, _ := url.Parse(got)
	q := u.Query()
	if q.Get("page") != "1" {
		t.Errorf("page = %q, want %q", q.Get("page"), "1")
	}
	if _, ok := q["per_page"]; ok {
		t.Errorf("per_page must be omitted when PerPage is 0, got query %q", u.RawQuery)
	}
}

func TestAddOptions_PerPageSet(t *testing.T) {
	got, err := addOptions("repos/o/r/issues", &ListOptions{Page: 2, PerPage: 50})
	if err != nil {
		t.Fatalf("addOptions returned error: %v", err)
	}
	u, _ := url.Parse(got)
	q := u.Query()
	if q.Get("page") != "2" || q.Get("per_page") != "50" {
		t.Errorf("got query %q, want page=2&per_page=50", u.RawQuery)
	}
}

func TestAddOptions_PerPageZeroPreservesExistingPerPage(t *testing.T) {
	// Rebuilding a URL that already carries a server-provided per_page
	// (e.g. copied from a Link header) must not drop it when the options
	// struct leaves PerPage at 0.
	got, err := addOptions("repos/o/r/issues?per_page=100", &ListOptions{Page: 2, PerPage: 0})
	if err != nil {
		t.Fatalf("addOptions returned error: %v", err)
	}
	u, _ := url.Parse(got)
	q := u.Query()
	if q.Get("per_page") != "100" {
		t.Errorf("per_page = %q, want %q (existing value preserved)", q.Get("per_page"), "100")
	}
	if q.Get("page") != "2" {
		t.Errorf("page = %q, want %q", q.Get("page"), "2")
	}
}

func TestAddOptions_EmbeddedListOptions(t *testing.T) {
	opts := &IssueListByRepoOptions{
		State:       "open",
		ListOptions: ListOptions{Page: 3, PerPage: 0},
	}
	got, err := addOptions("repos/o/r/issues", opts)
	if err != nil {
		t.Fatalf("addOptions returned error: %v", err)
	}
	u, _ := url.Parse(got)
	q := u.Query()
	if q.Get("state") != "open" || q.Get("page") != "3" {
		t.Errorf("got query %q, want state=open&page=3", u.RawQuery)
	}
	if _, ok := q["per_page"]; ok {
		t.Errorf("per_page must be omitted when PerPage is 0, got query %q", u.RawQuery)
	}
}

func responseWithLink(links ...string) *Response {
	h := http.Header{}
	for _, l := range links {
		h.Add("Link", l)
	}
	return newResponse(&http.Response{Header: h})
}

func TestParsePagination_WithoutPerPage(t *testing.T) {
	// Link header as returned when the request omitted per_page.
	resp := responseWithLink(`<https://api.github.com/repos/o/r/issues?page=3>; rel="next", <https://api.github.com/repos/o/r/issues?page=5>; rel="last", <https://api.github.com/repos/o/r/issues?page=1>; rel="first", <https://api.github.com/repos/o/r/issues?page=1>; rel="prev"`)
	if resp.NextPage != 3 || resp.LastPage != 5 || resp.FirstPage != 1 || resp.PrevPage != 1 {
		t.Errorf("got next=%d last=%d first=%d prev=%d, want 3/5/1/1",
			resp.NextPage, resp.LastPage, resp.FirstPage, resp.PrevPage)
	}
	if resp.PerPage != 0 {
		t.Errorf("PerPage = %d, want 0 (server did not advertise one)", resp.PerPage)
	}
}

func TestParsePagination_WithPerPage(t *testing.T) {
	resp := responseWithLink(`<https://api.github.com/repos/o/r/issues?page=2&per_page=50>; rel="next", <https://api.github.com/repos/o/r/issues?page=4&per_page=50>; rel="last"`)
	if resp.NextPage != 2 || resp.LastPage != 4 {
		t.Errorf("got next=%d last=%d, want 2/4", resp.NextPage, resp.LastPage)
	}
	if resp.PerPage != 50 {
		t.Errorf("PerPage = %d, want 50 (server-selected page size)", resp.PerPage)
	}
}

func TestParsePagination_MultipleHeaders(t *testing.T) {
	resp := responseWithLink(
		`<https://api.github.com/repos/o/r/issues?page=2&per_page=2>; rel="next"`,
		`<https://api.github.com/repos/o/r/issues?page=9&per_page=2>; rel="last"`,
	)
	if resp.NextPage != 2 || resp.LastPage != 9 || resp.PerPage != 2 {
		t.Errorf("got next=%d last=%d perPage=%d, want 2/9/2", resp.NextPage, resp.LastPage, resp.PerPage)
	}
}

func TestNextPageOptions_CarriesServerPerPage(t *testing.T) {
	resp := responseWithLink(`<https://api.github.com/repos/o/r/issues?page=2&per_page=2>; rel="next"`)
	next := resp.NextPageOptions(&ListOptions{Page: 1, PerPage: 0})
	if next == nil {
		t.Fatal("NextPageOptions returned nil, want options for page 2")
	}
	if next.Page != 2 || next.PerPage != 2 {
		t.Errorf("got page=%d perPage=%d, want 2/2", next.Page, next.PerPage)
	}
}

func TestIssuesListByRepo_TwoPageFlow(t *testing.T) {
	var queries []url.Values
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/o/r/issues" {
			http.NotFound(w, r)
			return
		}
		queries = append(queries, r.URL.Query())
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Query().Get("page") {
		case "", "1":
			w.Header().Set("Link",
				fmt.Sprintf(`<%s/repos/o/r/issues?page=2&per_page=2>; rel="next", <%s/repos/o/r/issues?page=2&per_page=2>; rel="last"`, srv.URL, srv.URL))
			io.WriteString(w, `[{"number":1,"title":"a"},{"number":2,"title":"b"}]`)
		case "2":
			io.WriteString(w, `[{"number":3,"title":"c"}]`)
		default:
			t.Errorf("unexpected page query: %q", r.URL.Query().Get("page"))
		}
	}))
	defer srv.Close()

	client := NewClient(nil)
	base, _ := url.Parse(srv.URL + "/")
	client.BaseURL = base

	ctx := context.Background()
	opts := &IssueListByRepoOptions{ListOptions: ListOptions{Page: 1, PerPage: 0}}
	page1, resp, err := client.Issues.ListByRepo(ctx, "o", "r", opts)
	if err != nil {
		t.Fatalf("ListByRepo page 1: %v", err)
	}
	if len(page1) != 2 {
		t.Fatalf("page 1 returned %d issues, want 2", len(page1))
	}
	if _, ok := queries[0]["per_page"]; ok {
		t.Errorf("first request must omit per_page when PerPage is 0, got %q", queries[0].Encode())
	}
	if resp.NextPage != 2 || resp.PerPage != 2 {
		t.Fatalf("got NextPage=%d PerPage=%d, want 2/2 (from Link header)", resp.NextPage, resp.PerPage)
	}

	nextOpts := &IssueListByRepoOptions{ListOptions: *resp.NextPageOptions(&opts.ListOptions)}
	page2, _, err := client.Issues.ListByRepo(ctx, "o", "r", nextOpts)
	if err != nil {
		t.Fatalf("ListByRepo page 2: %v", err)
	}
	if len(page2) != 1 {
		t.Fatalf("page 2 returned %d issues, want 1", len(page2))
	}
	if queries[1].Get("page") != "2" || queries[1].Get("per_page") != "2" {
		t.Errorf("second request query = %q, want page=2&per_page=2 (server page size preserved)", queries[1].Encode())
	}
}
