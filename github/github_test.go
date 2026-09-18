package github

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

func TestAddOptions_PerPageZeroOmitsQueryParam(t *testing.T) {
	t.Parallel()

	got, err := addOptions("https://api.github.com/repos", ListOptions{Page: 1, PerPage: 0})
	if err != nil {
		t.Fatalf("addOptions: %v", err)
	}

	u, err := url.Parse(got)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	q := u.Query()
	if q.Get("page") != "1" {
		t.Errorf("page=%q, want 1", q.Get("page"))
	}
	if _, ok := q["per_page"]; ok {
		t.Errorf("per_page should be omitted when PerPage is 0, got %q", q.Get("per_page"))
	}
	if strings.Contains(got, "per_page=0") {
		t.Errorf("URL %q must not contain per_page=0", got)
	}
}

func TestAddOptions_PerPagePositive(t *testing.T) {
	t.Parallel()

	got, err := addOptions("https://api.github.com/repos", &ListOptions{Page: 2, PerPage: 50})
	if err != nil {
		t.Fatalf("addOptions: %v", err)
	}

	q, err := url.ParseQuery(strings.SplitN(got, "?", 2)[1])
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if q.Get("page") != "2" || q.Get("per_page") != "50" {
		t.Errorf("got %q, want page=2 per_page=50", got)
	}
}

func TestAddOptions_NilAndZeroValue(t *testing.T) {
	t.Parallel()

	base := "https://api.github.com/user/repos"
	got, err := addOptions(base, (*ListOptions)(nil))
	if err != nil {
		t.Fatalf("nil pointer: %v", err)
	}
	if got != base {
		t.Errorf("nil pointer mutated URL: got %q want %q", got, base)
	}

	got, err = addOptions(base, ListOptions{})
	if err != nil {
		t.Fatalf("zero value: %v", err)
	}
	if strings.Contains(got, "?") {
		t.Errorf("zero ListOptions should not add a query string, got %q", got)
	}
}

func TestAddOptions_PreservesExistingQueryWhenPerPageZero(t *testing.T) {
	t.Parallel()

	base := "https://api.github.com/user/repos?type=owner&per_page=30"
	got, err := addOptions(base, ListOptions{Page: 2, PerPage: 0})
	if err != nil {
		t.Fatalf("addOptions: %v", err)
	}

	u, err := url.Parse(got)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	q := u.Query()
	if q.Get("type") != "owner" {
		t.Errorf("lost type=%q", q.Get("type"))
	}
	if q.Get("per_page") != "30" {
		t.Errorf("PerPage=0 should preserve existing per_page, got %q", q.Get("per_page"))
	}
	if q.Get("page") != "2" {
		t.Errorf("page=%q, want 2", q.Get("page"))
	}
}

func TestAddOptions_EmbeddedListOptions(t *testing.T) {
	t.Parallel()

	type repoListOptions struct {
		Type string `url:"type,omitempty"`
		ListOptions
	}

	got, err := addOptions("https://api.github.com/user/repos", repoListOptions{
		Type:        "public",
		ListOptions: ListOptions{Page: 1, PerPage: 0},
	})
	if err != nil {
		t.Fatalf("addOptions: %v", err)
	}
	if strings.Contains(got, "per_page") {
		t.Errorf("embedded PerPage=0 leaked per_page in %q", got)
	}
	if !strings.Contains(got, "type=public") || !strings.Contains(got, "page=1") {
		t.Errorf("missing expected params in %q", got)
	}
}

func TestBuildURL_PerPageZeroPreservesLinkContext(t *testing.T) {
	t.Parallel()

	base := "https://api.github.com/user/repos?page=1&per_page=30"
	got, err := buildURL(base, ListOptions{Page: 2, PerPage: 0})
	if err != nil {
		t.Fatalf("buildURL: %v", err)
	}

	u, err := url.Parse(got)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if u.Query().Get("page") != "2" {
		t.Errorf("page=%q, want 2", u.Query().Get("page"))
	}
	if u.Query().Get("per_page") != "30" {
		t.Errorf("per_page=%q, want 30 from previous URL", u.Query().Get("per_page"))
	}
}

func TestBuildURL_ExplicitPerPageOverwrites(t *testing.T) {
	t.Parallel()

	got, err := buildURL("https://api.github.com/repos?page=1", ListOptions{Page: 3, PerPage: 10})
	if err != nil {
		t.Fatalf("buildURL: %v", err)
	}
	u, _ := url.Parse(got)
	if u.Query().Get("page") != "3" || u.Query().Get("per_page") != "10" {
		t.Errorf("got %q", got)
	}
}

func TestParsePagination_PageOnlyLinkHeader(t *testing.T) {
	t.Parallel()

	httpResp := &http.Response{Header: make(http.Header)}
	httpResp.Header.Set("Link", strings.Join([]string{
		`<https://api.github.com/user/repos?page=1>; rel="first"`,
		`<https://api.github.com/user/repos?page=2>; rel="prev"`,
		`<https://api.github.com/user/repos?page=4>; rel="next"`,
		`<https://api.github.com/user/repos?page=5>; rel="last"`,
	}, ", "))

	resp := newResponse(httpResp)
	if resp.FirstPage != 1 || resp.PrevPage != 2 || resp.NextPage != 4 || resp.LastPage != 5 {
		t.Fatalf("pages first=%d prev=%d next=%d last=%d", resp.FirstPage, resp.PrevPage, resp.NextPage, resp.LastPage)
	}
	if resp.PerPage != 0 {
		t.Errorf("PerPage=%d, want 0 when Link omits per_page", resp.PerPage)
	}
}

func TestParsePagination_PageAndPerPageLinkHeader(t *testing.T) {
	t.Parallel()

	httpResp := &http.Response{Header: make(http.Header)}
	httpResp.Header.Set("Link", strings.Join([]string{
		`<https://api.github.com/user/repos?page=1&per_page=30>; rel="first"`,
		`<https://api.github.com/user/repos?page=2&per_page=30>; rel="next"`,
		`<https://api.github.com/user/repos?page=4&per_page=30>; rel="last"`,
	}, ", "))

	resp := newResponse(httpResp)
	if resp.FirstPage != 1 || resp.NextPage != 2 || resp.LastPage != 4 {
		t.Fatalf("pages first=%d next=%d last=%d", resp.FirstPage, resp.NextPage, resp.LastPage)
	}
	if resp.PerPage != 30 {
		t.Errorf("PerPage=%d, want 30 from Link query", resp.PerPage)
	}

	next := resp.NextPageOptions(ListOptions{PerPage: 0})
	if next != (ListOptions{Page: 2, PerPage: 30}) {
		t.Errorf("NextPageOptions=%+v, want page=2 per_page=30", next)
	}
}

func TestParsePagination_InvalidAndEmpty(t *testing.T) {
	t.Parallel()

	httpResp := &http.Response{Header: make(http.Header)}
	httpResp.Header.Set("Link", strings.Join([]string{
		`,`,
		`https://api.github.com/?page=2; rel="prev"`,
		`<https://api.github.com/?q=nope>; rel="next"`,
		`<https://api.github.com/?page=notanumber>; rel="last"`,
		`<https://api.github.com/?page=3>; rel="next"`,
	}, ", "))

	resp := newResponse(httpResp)
	if resp.NextPage != 3 {
		t.Errorf("NextPage=%d, want 3 from the only valid link", resp.NextPage)
	}
	if resp.PrevPage != 0 || resp.LastPage != 0 {
		t.Errorf("invalid links should be ignored, prev=%d last=%d", resp.PrevPage, resp.LastPage)
	}

	empty := newResponse(&http.Response{Header: make(http.Header)})
	if empty.NextPage != 0 {
		t.Errorf("empty Link should leave pages at 0")
	}
}

func TestResponse_PageOptionsPreserveExplicitPerPage(t *testing.T) {
	t.Parallel()

	resp := &Response{NextPage: 2, PrevPage: 1, FirstPage: 1, LastPage: 5, PerPage: 30}
	got := resp.NextPageOptions(ListOptions{Page: 1, PerPage: 10})
	if got != (ListOptions{Page: 2, PerPage: 10}) {
		t.Errorf("explicit PerPage should win, got %+v", got)
	}
}

func TestClient_MultiPageTraversalWithPerPageZero(t *testing.T) {
	t.Parallel()

	const implicitSize = 30
	pages := map[int][]string{
		1: {"a", "b"},
		2: {"c", "d"},
		3: {"e"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("per_page") == "0" {
			t.Errorf("server received invalid per_page=0 on %s", r.URL.RawQuery)
		}

		page := 1
		if v := r.URL.Query().Get("page"); v != "" {
			var err error
			page, err = strconv.Atoi(v)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}

		items, ok := pages[page]
		if !ok {
			http.NotFound(w, r)
			return
		}

		var links []string
		linkFor := func(p int, rel string) string {
			u := *r.URL
			q := u.Query()
			q.Set("page", strconv.Itoa(p))
			q.Set("per_page", strconv.Itoa(implicitSize))
			u.RawQuery = q.Encode()
			return `<` + "http://" + r.Host + u.String() + `>; rel="` + rel + `"`
		}
		if page > 1 {
			links = append(links, linkFor(page-1, "prev"), linkFor(1, "first"))
		}
		if _, hasNext := pages[page+1]; hasNext {
			links = append(links, linkFor(page+1, "next"), linkFor(len(pages), "last"))
		}
		if len(links) > 0 {
			w.Header().Set("Link", strings.Join(links, ", "))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(items)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, server.Client())
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	opts := &ListOptions{Page: 1, PerPage: 0}
	var seen []string
	visited := 0
	var resp *Response

	for {
		var body []byte
		var listErr error
		if resp == nil {
			resp, body, listErr = client.List(context.Background(), "/user/repos", opts)
		} else {
			resp, body, listErr = client.NextPage(context.Background(), "/user/repos", resp, opts)
			if resp == nil && listErr == nil {
				break
			}
		}
		if listErr != nil {
			t.Fatalf("list: %v", listErr)
		}

		var batch []string
		if err := json.Unmarshal(body, &batch); err != nil {
			t.Fatalf("decode: %v", err)
		}
		seen = append(seen, batch...)
		visited++

		if resp.NextPage != 0 && resp.PerPage != implicitSize {
			t.Fatalf("page %d: Link per_page=%d, want %d so follow-up requests stay consistent", visited, resp.PerPage, implicitSize)
		}

		follow := resp.NextPageOptions(*opts)
		if resp.NextPage != 0 && follow.PerPage != implicitSize {
			t.Fatalf("follow-up options %+v did not pin per_page from Link", follow)
		}

		opts = &ListOptions{Page: resp.NextPage, PerPage: 0}
		if resp.NextPage == 0 {
			break
		}
	}

	if visited != 3 {
		t.Fatalf("visited %d pages, want 3", visited)
	}
	want := []string{"a", "b", "c", "d", "e"}
	if strings.Join(seen, ",") != strings.Join(want, ",") {
		t.Fatalf("items=%v, want %v", seen, want)
	}
}

func TestClient_FollowPageUsesLinkPerPage(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("per_page") == "0" {
			t.Errorf("invalid per_page=0")
		}
		page := r.URL.Query().Get("page")
		if page == "2" && r.URL.Query().Get("per_page") != "30" {
			t.Errorf("page 2 missing preserved per_page, query=%q", r.URL.RawQuery)
		}
		w.Header().Set("Link", `<http://example.com/items?page=2&per_page=30>; rel="next"`)
		io.WriteString(w, `[]`)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, server.Client())
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	first, _, err := client.List(context.Background(), "/items", &ListOptions{PerPage: 0})
	if err != nil {
		t.Fatalf("first page: %v", err)
	}
	if _, _, err := client.FollowPage(context.Background(), "/items", first.NextPage, first, &ListOptions{PerPage: 0}); err != nil {
		t.Fatalf("follow: %v", err)
	}
}
