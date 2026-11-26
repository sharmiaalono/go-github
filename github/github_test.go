package github

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
)

func TestAddOptionsPerPageZeroAndExplicit(t *testing.T) {
	t.Parallel()

	implicit, err := addOptions(
		"repos/acme/widgets/issues?labels=security",
		&IssueListByRepoOptions{State: "open", ListOptions: ListOptions{Page: 1}},
	)
	if err != nil {
		t.Fatalf("addOptions implicit: %v", err)
	}
	assertQuery(t, implicit, map[string]string{
		"labels":   "security",
		"page":     "1",
		"per_page": "",
		"state":    "open",
	})

	explicit, err := addOptions(
		"repos/acme/widgets/issues",
		&IssueListByRepoOptions{State: "open", ListOptions: ListOptions{Page: 2, PerPage: 50}},
	)
	if err != nil {
		t.Fatalf("addOptions explicit: %v", err)
	}
	assertQuery(t, explicit, map[string]string{
		"page":     "2",
		"per_page": "50",
		"state":    "open",
	})
}

func TestAddOptionsPreservesLinkedQueryAndDoesNotMutateCaller(t *testing.T) {
	t.Parallel()

	original := &IssueListByRepoOptions{
		ListOptions: ListOptions{Page: 2},
		State:       "all",
	}
	got, err := addOptions(
		"repos/acme/widgets/issues?labels=bug&per_page=40&state=open",
		original,
	)
	if err != nil {
		t.Fatalf("addOptions: %v", err)
	}
	assertQuery(t, got, map[string]string{
		"labels":   "bug",
		"page":     "2",
		"per_page": "40",
		"state":    "all",
	})
	if original.Page != 2 || original.PerPage != 0 || original.State != "all" {
		t.Fatalf("caller options were mutated: %+v", original)
	}
}

func TestParseLinkHeaderWithoutPerPage(t *testing.T) {
	t.Parallel()

	response := newResponse(&http.Response{Header: http.Header{
		"Link": []string{
			`<https://api.github.com/repos/o/r/issues?page=2>; rel="next", ` +
				`<https://api.github.com/repos/o/r/issues?page=6>; rel="last"`,
			`<https://api.github.com/repos/o/r/issues?page=1>; rel="first prev"`,
		},
	}})

	if response.NextPage != 2 || response.LastPage != 6 || response.FirstPage != 1 || response.PrevPage != 1 {
		t.Fatalf("unexpected page fields: %+v", response)
	}
	if response.PerPage != DefaultPerPage {
		t.Fatalf("PerPage = %d, want default %d", response.PerPage, DefaultPerPage)
	}
}

func TestParseLinkHeaderRetainsServerPerPage(t *testing.T) {
	t.Parallel()

	response := newResponse(&http.Response{Header: http.Header{
		"Link": []string{`<https://api.github.com/repos/o/r/issues?per_page=75&page=3>; rel="next"`},
	}})

	next, ok := response.NextPageOptions(ListOptions{Page: 2})
	if !ok {
		t.Fatal("NextPageOptions reported no next page")
	}
	if next.Page != 3 || next.PerPage != 75 {
		t.Fatalf("NextPageOptions = %+v, want page=3 per_page=75", next)
	}
}

func TestRequestNeverSendsPerPageZero(t *testing.T) {
	t.Parallel()

	value, err := addOptions("repos/o/r/issues", &ListOptions{Page: 1, PerPage: 0})
	if err != nil {
		t.Fatalf("addOptions: %v", err)
	}
	parsed, err := url.Parse(value)
	if err != nil {
		t.Fatalf("parse URL: %v", err)
	}
	if parsed.Query().Has("per_page") {
		t.Fatalf("PerPage=0 emitted per_page: %s", value)
	}
}

func TestListFollowsLinkHeadersAndPreservesFilters(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		query := request.URL.Query()
		if query.Get("labels") != "security" || query.Get("state") != "open" {
			t.Errorf("filters lost from query: %s", request.URL.RawQuery)
		}
		if query.Get("per_page") == "0" {
			t.Errorf("per_page=0 must never be sent: %s", request.URL.RawQuery)
		}

		switch query.Get("page") {
		case "", "1":
			writer.Header().Set("Link", fmt.Sprintf(
				`<%s/repos/acme/widgets/issues?labels=security&state=open&per_page=40&page=2>; rel="next", `+
					`<%s/repos/acme/widgets/issues?labels=security&state=open&per_page=40&page=2>; rel="last"`,
				server.URL, server.URL,
			))
			_, _ = writer.Write([]byte(`[{"number":1,"title":"first"}]`))
		case "2":
			if query.Get("per_page") != "40" {
				t.Errorf("server page size was not preserved: %s", request.URL.RawQuery)
			}
			_, _ = writer.Write([]byte(`[{"number":2,"title":"second"}]`))
		default:
			http.Error(writer, "unexpected page", http.StatusBadRequest)
		}
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.BaseURL, _ = url.Parse(server.URL + "/")
	options := &IssueListByRepoOptions{
		Labels: "security",
		State:  "open",
	}

	var issues []*Issue
	for {
		page, response, err := client.Issues.ListByRepo(context.Background(), "acme", "widgets", options)
		if err != nil {
			t.Fatalf("ListByRepo: %v", err)
		}
		issues = append(issues, page...)
		next, ok := response.NextPageOptions(options.ListOptions)
		if !ok {
			break
		}
		options.ListOptions = next
	}

	if requests.Load() != 2 {
		t.Fatalf("made %d requests, want 2", requests.Load())
	}
	if len(issues) != 2 || issues[0].Number != 1 || issues[1].Number != 2 {
		t.Fatalf("issues across pages = %+v", issues)
	}
}

func assertQuery(t *testing.T, rawURL string, expected map[string]string) {
	t.Helper()
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse %q: %v", rawURL, err)
	}
	query := parsed.Query()
	for key, value := range expected {
		if got := query.Get(key); got != value {
			t.Errorf("query %s = %q, want %q (full URL %s)", key, got, value, rawURL)
		}
	}
	if strings.Contains(query.Encode(), "per_page=0") {
		t.Errorf("query contains per_page=0: %s", rawURL)
	}
}
