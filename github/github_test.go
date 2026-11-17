package github

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

var DQ_STR = string(rune(34))

func TestParsePagination_PerPageZeroAndExplicit(t *testing.T) {
	links := []string{
		`<https://api.github.com/repos/o/r/issues?page=2>; rel="next", <https://api.github.com/repos/o/r/issues?page=6>; rel="last", <https://api.github.com/repos/o/r/issues?page=1>; rel="first", <https://api.github.com/repos/o/r/issues?page=1>; rel="prev"`,
	}
	hdr := http.Header{"Link": links}
	resp := NewResponse(&http.Response{Header: hdr})
	if resp.NextPage != 2 || resp.LastPage != 6 || resp.FirstPage != 1 || resp.PrevPage != 1 {
		t.Fatalf("unexpected pages: %+v", resp)
	}
	if resp.PerPage != DefaultPerPage {
		t.Fatalf("expected DefaultPerPage=%d, got %d", DefaultPerPage, resp.PerPage)
	}
}

func TestAddOptions_ZeroVsExplicitPerPageAndFilterRetention(t *testing.T) {
	u1, err := addOptions("repos/o/r/issues?labels=bounty", &IssueListByRepoOptions{State: "open", ListOptions: ListOptions{Page: 1, PerPage: 0}})
	if err != nil { t.Fatalf("unexpected err: %v", err) }
	p1, _ := url.Parse(u1)
	q1 := p1.Query()
	if q1.Get("page") != "1" || q1.Get("per_page") != "" || q1.Get("state") != "open" || q1.Get("labels") != "bounty" {
		t.Fatalf("unexpected query for PerPage=0: %s", u1)
	}
	u2, err := addOptions("repos/o/r/issues?per_page=50&state=open", &ListOptions{Page: 2, PerPage: 0})
	if err != nil { t.Fatalf("unexpected err: %v", err) }
	p2, _ := url.Parse(u2)
	q2 := p2.Query()
	if q2.Get("page") != "2" || q2.Get("per_page") != "50" || q2.Get("state") != "open" {
		t.Fatalf("expected server per_page=50 preserved when PerPage=0, got %s", u2)
	}
	u3, err := addOptions("repos/o/r/issues", &ListOptions{Page: 2, PerPage: 50})
	if err != nil { t.Fatalf("unexpected err: %v", err) }
	p3, _ := url.Parse(u3)
	if p3.Query().Get("page") != "2" || p3.Query().Get("per_page") != "50" {
		t.Fatalf("expected explicit page=2&per_page=50, got %s", u3)
	}
}

func TestMultiPageFlow_IssuesAndRepos_HttpTestServer(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("state") != "open" { t.Errorf("expected state=open retained, got %s", r.URL.RawQuery) }
		if q.Get("per_page") == "0" { t.Errorf("must never send per_page=0, got %s", r.URL.RawQuery) }
		page := q.Get("page")
		if page == "" || page == "1" {
			w.Header().Set("Link", fmt.Sprintf("<%s/repos/o/r/issues?state=open&per_page=40&page=2>; rel=%snext%s, <%s/repos/o/r/issues?state=open&per_page=40&page=2>; rel=%slast%s", srv.URL, DQ_STR, DQ_STR, srv.URL, DQ_STR, DQ_STR))
			_, _ = w.Write([]byte(`[{"number":1,"title":"first"}]`))
			return
		}
		if q.Get("per_page") != "40" { t.Errorf("expected per_page=40 preserved, got %s", r.URL.RawQuery) }
		w.Header().Set("Link", fmt.Sprintf("<%s/repos/o/r/issues?state=open&per_page=40&page=1>; rel=%sprev%s", srv.URL, DQ_STR, DQ_STR))
		_, _ = w.Write([]byte(`[{"number":2,"title":"second"}]`))
	}))
	defer srv.Close()
	client := NewClient(srv.Client())
	base, _ := url.Parse(srv.URL + "/")
	client.BaseURL = base
	opts := &IssueListByRepoOptions{State: "open", ListOptions: ListOptions{PerPage: 0}}
	all := []*Issue{}
	for {
		items, resp, err := client.Issues.ListByRepo("o", "r", opts)
		if err != nil { t.Fatalf("ListByRepo error: %v", err) }
		all = append(all, items...)
		nextOpts, ok := resp.NextPageOptions(opts.ListOptions)
		if !ok { break }
		opts.ListOptions = nextOpts
	}
	if len(all) != 2 || all[0].Number != 1 || all[1].Number != 2 {
		t.Fatalf("expected 2 issues across paginated calls, got %+v", all)
	}
}
