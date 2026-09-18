package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
)

func TestAddOptions(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		opts    any
		wantURL string
		wantErr bool
	}{
		{
			name:    "ListOptions with Page 1 and PerPage 0 omits per_page",
			baseURL: "https://api.github.com/resource",
			opts:    ListOptions{Page: 1, PerPage: 0},
			wantURL: "https://api.github.com/resource?page=1",
		},
		{
			name:    "Pointer ListOptions with Page 1 and PerPage 0 omits per_page",
			baseURL: "https://api.github.com/resource",
			opts:    &ListOptions{Page: 1, PerPage: 0},
			wantURL: "https://api.github.com/resource?page=1",
		},
		{
			name:    "ListOptions with both Page and PerPage set",
			baseURL: "https://api.github.com/resource",
			opts:    &ListOptions{Page: 2, PerPage: 50},
			wantURL: "https://api.github.com/resource?page=2&per_page=50",
		},
		{
			name:    "ListOptions with zero Page and zero PerPage produces base URL",
			baseURL: "https://api.github.com/resource",
			opts:    &ListOptions{Page: 0, PerPage: 0},
			wantURL: "https://api.github.com/resource",
		},
		{
			name:    "Preserves existing query parameters on base URL",
			baseURL: "https://api.github.com/resource?sort=updated",
			opts:    &ListOptions{Page: 1, PerPage: 0},
			wantURL: "https://api.github.com/resource?page=1&sort=updated",
		},
		{
			name:    "Preserves existing per_page if options has PerPage 0",
			baseURL: "https://api.github.com/resource?per_page=30",
			opts:    &ListOptions{Page: 2, PerPage: 0},
			wantURL: "https://api.github.com/resource?page=2&per_page=30",
		},
		{
			name:    "Overrides existing per_page when explicit PerPage is provided",
			baseURL: "https://api.github.com/resource?per_page=30",
			opts:    &ListOptions{Page: 2, PerPage: 100},
			wantURL: "https://api.github.com/resource?page=2&per_page=100",
		},
		{
			name:    "RepositoryListOptions embedding ListOptions with PerPage 0",
			baseURL: "https://api.github.com/user/repos",
			opts: &RepositoryListOptions{
				Visibility:  "public",
				Sort:        "created",
				Direction:   "desc",
				ListOptions: ListOptions{Page: 3, PerPage: 0},
			},
			wantURL: "https://api.github.com/user/repos?direction=desc&page=3&sort=created&visibility=public",
		},
		{
			name:    "IssueListByRepoOptions embedding ListOptions with PerPage 25",
			baseURL: "https://api.github.com/repos/octocat/hello-world/issues",
			opts: &IssueListByRepoOptions{
				State:       "open",
				ListOptions: ListOptions{Page: 1, PerPage: 25},
			},
			wantURL: "https://api.github.com/repos/octocat/hello-world/issues?page=1&per_page=25&state=open",
		},
		{
			name:    "Nil options returns original URL unmodified",
			baseURL: "https://api.github.com/resource",
			opts:    nil,
			wantURL: "https://api.github.com/resource",
		},
		{
			name:    "Malformed base URL returns error",
			baseURL: "http://[::1]:namedport",
			opts:    &ListOptions{Page: 1},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := AddOptions(tt.baseURL, tt.opts)
			if (err != nil) != tt.wantErr {
				t.Fatalf("AddOptions() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			gotParsed, err := url.Parse(got)
			if err != nil {
				t.Fatalf("Failed to parse result URL %q: %v", got, err)
			}
			wantParsed, err := url.Parse(tt.wantURL)
			if err != nil {
				t.Fatalf("Failed to parse expected URL %q: %v", tt.wantURL, err)
			}
			if gotParsed.Path != wantParsed.Path || gotParsed.Host != wantParsed.Host {
				t.Errorf("AddOptions() path/host mismatch: got %v, want %v", got, tt.wantURL)
			}
			if !reflect.DeepEqual(gotParsed.Query(), wantParsed.Query()) {
				t.Errorf("AddOptions() query mismatch: got %v, want %v", gotParsed.Query(), wantParsed.Query())
			}
		})
	}
}

func TestBuildURL(t *testing.T) {
	tests := []struct {
		name    string
		rawURL  string
		opts    *ListOptions
		wantURL string
		wantErr bool
	}{
		{
			name:    "BuildURL with PerPage 0 leaves per_page unset",
			rawURL:  "https://api.github.com/resource",
			opts:    &ListOptions{Page: 1, PerPage: 0},
			wantURL: "https://api.github.com/resource?page=1",
		},
		{
			name:    "BuildURL preserves existing per_page when PerPage is 0",
			rawURL:  "https://api.github.com/resource?per_page=30",
			opts:    &ListOptions{Page: 2, PerPage: 0},
			wantURL: "https://api.github.com/resource?page=2&per_page=30",
		},
		{
			name:    "BuildURL overrides per_page when PerPage is positive",
			rawURL:  "https://api.github.com/resource?per_page=30",
			opts:    &ListOptions{Page: 2, PerPage: 50},
			wantURL: "https://api.github.com/resource?page=2&per_page=50",
		},
		{
			name:    "BuildURL with nil options returns original URL",
			rawURL:  "https://api.github.com/resource",
			opts:    nil,
			wantURL: "https://api.github.com/resource",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildURL(tt.rawURL, tt.opts)
			if (err != nil) != tt.wantErr {
				t.Fatalf("BuildURL() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			gotParsed, _ := url.Parse(got)
			wantParsed, _ := url.Parse(tt.wantURL)
			if !reflect.DeepEqual(gotParsed.Query(), wantParsed.Query()) {
				t.Errorf("BuildURL() query mismatch: got %v, want %v", gotParsed.Query(), wantParsed.Query())
			}
		})
	}
}

func TestParsePagination_OnlyPageParameters(t *testing.T) {
	rawResp := &http.Response{
		Header: http.Header{
			"Link": {
				`<https://api.github.com/repos?page=1>; rel="first", ` +
					`<https://api.github.com/repos?page=2>; rel="prev", ` +
					`<https://api.github.com/repos?page=4>; rel="next", ` +
					`<https://api.github.com/repos?page=5>; rel="last"`,
			},
		},
	}

	resp := newResponse(rawResp)

	if resp.FirstPage != 1 {
		t.Errorf("resp.FirstPage = %d, want 1", resp.FirstPage)
	}
	if resp.PrevPage != 2 {
		t.Errorf("resp.PrevPage = %d, want 2", resp.PrevPage)
	}
	if resp.NextPage != 4 {
		t.Errorf("resp.NextPage = %d, want 4", resp.NextPage)
	}
	if resp.LastPage != 5 {
		t.Errorf("resp.LastPage = %d, want 5", resp.LastPage)
	}
	if resp.PerPage != 0 {
		t.Errorf("resp.PerPage = %d, want 0 when omitted from Link", resp.PerPage)
	}

	nextOpts := resp.NextPageOptions()
	if nextOpts == nil {
		t.Fatal("resp.NextPageOptions() is nil, expected non-nil")
	}
	if nextOpts.Page != 4 || nextOpts.PerPage != 0 {
		t.Errorf("nextOpts = %+v, want Page 4, PerPage 0", nextOpts)
	}
}

func TestParsePagination_PageAndPerPageParameters(t *testing.T) {
	rawResp := &http.Response{
		Header: http.Header{
			"Link": {
				`<https://api.github.com/repos?page=1&per_page=25>; rel="first", ` +
					`<https://api.github.com/repos?page=2&per_page=25>; rel="prev", ` +
					`<https://api.github.com/repos?page=4&per_page=25>; rel="next", ` +
					`<https://api.github.com/repos?page=5&per_page=25>; rel="last"`,
			},
		},
	}

	resp := newResponse(rawResp)

	if resp.FirstPage != 1 {
		t.Errorf("resp.FirstPage = %d, want 1", resp.FirstPage)
	}
	if resp.PrevPage != 2 {
		t.Errorf("resp.PrevPage = %d, want 2", resp.PrevPage)
	}
	if resp.NextPage != 4 {
		t.Errorf("resp.NextPage = %d, want 4", resp.NextPage)
	}
	if resp.LastPage != 5 {
		t.Errorf("resp.LastPage = %d, want 5", resp.LastPage)
	}
	if resp.PerPage != 25 {
		t.Errorf("resp.PerPage = %d, want 25", resp.PerPage)
	}

	nextOpts := resp.NextPageOptions()
	if nextOpts == nil {
		t.Fatal("resp.NextPageOptions() is nil, expected non-nil")
	}
	if nextOpts.Page != 4 || nextOpts.PerPage != 25 {
		t.Errorf("nextOpts = %+v, want Page 4, PerPage 25", nextOpts)
	}
}

func TestParsePagination_CursorAndTokens(t *testing.T) {
	rawResp := &http.Response{
		Header: http.Header{
			"Link": {
				`<https://api.github.com/repos?cursor=cursor_token_xyz>; rel="next"`,
			},
		},
	}

	resp := newResponse(rawResp)
	if resp.Cursor != "cursor_token_xyz" {
		t.Errorf("resp.Cursor = %q, want %q", resp.Cursor, "cursor_token_xyz")
	}
}

func TestParsePagination_MalformedHeaders(t *testing.T) {
	malformed := []*http.Response{
		nil,
		{Header: http.Header{}},
		{Header: http.Header{"Link": {"not-a-valid-link-header"}}},
		{Header: http.Header{"Link": {"<invalid url>; rel=next"}}},
		{Header: http.Header{"Link": {"<https://api.github.com/?page=>; rel=\"next\""}}},
	}

	for i, r := range malformed {
		resp := &Response{Response: r}
		ParsePagination(resp)
		if resp.NextPage != 0 {
			t.Errorf("malformed test index %d unexpectedly set NextPage: %d", i, resp.NextPage)
		}
	}
}

func TestMultiPageTraversal_MockHTTP_Integration(t *testing.T) {
	type item struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	receivedQueries := make([]url.Values, 0)

	var serverURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQueries = append(receivedQueries, r.URL.Query())
		pageStr := r.URL.Query().Get("page")

		w.Header().Set("Content-Type", "application/json")

		switch pageStr {
		case "", "1":
			w.Header().Set("Link", fmt.Sprintf(
				`<%s/items?page=2>; rel="next", <%s/items?page=3>; rel="last"`,
				serverURL, serverURL,
			))
			json.NewEncoder(w).Encode([]item{
				{ID: 1, Name: "Item 1"},
				{ID: 2, Name: "Item 2"},
			})
		case "2":
			w.Header().Set("Link", fmt.Sprintf(
				`<%s/items?page=1>; rel="prev", <%s/items?page=3>; rel="next", <%s/items?page=3>; rel="last"`,
				serverURL, serverURL, serverURL,
			))
			json.NewEncoder(w).Encode([]item{
				{ID: 3, Name: "Item 3"},
				{ID: 4, Name: "Item 4"},
			})
		case "3":
			w.Header().Set("Link", fmt.Sprintf(
				`<%s/items?page=2>; rel="prev", <%s/items?page=1>; rel="first"`,
				serverURL, serverURL,
			))
			json.NewEncoder(w).Encode([]item{
				{ID: 5, Name: "Item 5"},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	serverURL = server.URL

	client := NewClient(server.Client())
	parsedBaseURL, err := url.Parse(server.URL + "/")
	if err != nil {
		t.Fatalf("Failed to parse test server URL: %v", err)
	}
	client.BaseURL = parsedBaseURL

	opts := &ListOptions{
		Page:    1,
		PerPage: 0,
	}

	var allItems []item
	ctx := context.Background()

	for {
		path, err := AddOptions("items", opts)
		if err != nil {
			t.Fatalf("AddOptions failed: %v", err)
		}

		req, err := client.NewRequest(http.MethodGet, path, nil)
		if err != nil {
			t.Fatalf("NewRequest failed: %v", err)
		}

		var pageItems []item
		resp, err := client.Do(ctx, req, &pageItems)
		if err != nil {
			t.Fatalf("client.Do failed on page %d: %v", opts.Page, err)
		}

		allItems = append(allItems, pageItems...)

		if resp.NextPage == 0 {
			break
		}

		opts.Page = resp.NextPage
	}

	if len(allItems) != 5 {
		t.Fatalf("Expected 5 items across 3 pages, got %d", len(allItems))
	}

	expectedIDs := []int{1, 2, 3, 4, 5}
	for i, it := range allItems {
		if it.ID != expectedIDs[i] {
			t.Errorf("Item %d: got ID %d, want %d", i, it.ID, expectedIDs[i])
		}
	}

	if len(receivedQueries) != 3 {
		t.Fatalf("Expected 3 HTTP requests, got %d", len(receivedQueries))
	}

	for i, q := range receivedQueries {
		if q.Get("per_page") != "" {
			t.Errorf("Request %d unexpectedly sent per_page query param: %v", i+1, q.Get("per_page"))
		}
		expectedPage := fmt.Sprintf("%d", i+1)
		if q.Get("page") != expectedPage {
			t.Errorf("Request %d sent page %q, want %q", i+1, q.Get("page"), expectedPage)
		}
	}
}

func TestMultiPageTraversal_RepositoriesList_Integration(t *testing.T) {
	var serverURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pageStr := r.URL.Query().Get("page")
		perPageStr := r.URL.Query().Get("per_page")

		if perPageStr != "" {
			t.Errorf("Unexpected per_page query parameter sent: %s", perPageStr)
		}

		w.Header().Set("Content-Type", "application/json")

		switch pageStr {
		case "", "1":
			w.Header().Set("Link", fmt.Sprintf(`<%s/users/octocat/repos?page=2>; rel="next"`, serverURL))
			json.NewEncoder(w).Encode([]*Repository{
				{ID: 101, Name: "repo-1"},
				{ID: 102, Name: "repo-2"},
			})
		case "2":
			w.Header().Set("Link", fmt.Sprintf(`<%s/users/octocat/repos?page=1>; rel="first"`, serverURL))
			json.NewEncoder(w).Encode([]*Repository{
				{ID: 103, Name: "repo-3"},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	serverURL = server.URL

	client := NewClient(server.Client())
	parsedBaseURL, _ := url.Parse(server.URL + "/")
	client.BaseURL = parsedBaseURL

	opts := &RepositoryListOptions{
		Type:        "owner",
		ListOptions: ListOptions{Page: 1, PerPage: 0},
	}

	ctx := context.Background()
	var allRepos []*Repository

	for {
		repos, resp, err := client.Repositories.List(ctx, "octocat", opts)
		if err != nil {
			t.Fatalf("Repositories.List failed on page %d: %v", opts.Page, err)
		}

		allRepos = append(allRepos, repos...)

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	if len(allRepos) != 3 {
		t.Fatalf("Expected 3 repositories, got %d", len(allRepos))
	}
}

func TestMultiPageTraversal_IssuesListByRepo_Integration(t *testing.T) {
	var serverURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pageStr := r.URL.Query().Get("page")
		stateStr := r.URL.Query().Get("state")

		if stateStr != "open" {
			t.Errorf("Expected state=open, got %s", stateStr)
		}

		w.Header().Set("Content-Type", "application/json")

		switch pageStr {
		case "", "1":
			w.Header().Set("Link", fmt.Sprintf(`<%s/repos/octocat/hello-world/issues?page=2&state=open>; rel="next"`, serverURL))
			json.NewEncoder(w).Encode([]*Issue{
				{ID: 1, Number: 10, Title: "Issue 10"},
			})
		case "2":
			w.Header().Set("Link", fmt.Sprintf(`<%s/repos/octocat/hello-world/issues?page=1&state=open>; rel="first"`, serverURL))
			json.NewEncoder(w).Encode([]*Issue{
				{ID: 2, Number: 20, Title: "Issue 20"},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	serverURL = server.URL

	client := NewClient(server.Client())
	parsedBaseURL, _ := url.Parse(server.URL + "/")
	client.BaseURL = parsedBaseURL

	opts := &IssueListByRepoOptions{
		State:       "open",
		ListOptions: ListOptions{Page: 1, PerPage: 0},
	}

	ctx := context.Background()
	var allIssues []*Issue

	for {
		issues, resp, err := client.Issues.ListByRepo(ctx, "octocat", "hello-world", opts)
		if err != nil {
			t.Fatalf("Issues.ListByRepo failed on page %d: %v", opts.Page, err)
		}

		allIssues = append(allIssues, issues...)

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	if len(allIssues) != 2 {
		t.Fatalf("Expected 2 issues, got %d", len(allIssues))
	}
}

func TestMultiPageTraversal_WithPerPageLinkPreservation_Integration(t *testing.T) {
	type item struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	var observedPerPages []string
	var serverURL string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		observedPerPages = append(observedPerPages, r.URL.Query().Get("per_page"))
		pageStr := r.URL.Query().Get("page")

		w.Header().Set("Content-Type", "application/json")

		switch pageStr {
		case "", "1":
			w.Header().Set("Link", fmt.Sprintf(
				`<%s/items?page=2&per_page=10>; rel="next", <%s/items?page=2&per_page=10>; rel="last"`,
				serverURL, serverURL,
			))
			json.NewEncoder(w).Encode([]item{{ID: 1, Name: "Item 1"}})
		case "2":
			w.Header().Set("Link", fmt.Sprintf(
				`<%s/items?page=1&per_page=10>; rel="first"`,
				serverURL,
			))
			json.NewEncoder(w).Encode([]item{{ID: 2, Name: "Item 2"}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	serverURL = server.URL

	client := NewClient(server.Client())
	parsedBaseURL, _ := url.Parse(server.URL + "/")
	client.BaseURL = parsedBaseURL

	opts := &ListOptions{
		Page:    1,
		PerPage: 0,
	}

	ctx := context.Background()

	path, err := AddOptions("items", opts)
	if err != nil {
		t.Fatalf("AddOptions failed: %v", err)
	}

	req, err := client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		t.Fatalf("NewRequest failed: %v", err)
	}

	var page1 []item
	resp, err := client.Do(ctx, req, &page1)
	if err != nil {
		t.Fatalf("First request failed: %v", err)
	}

	if resp.PerPage != 10 {
		t.Fatalf("resp.PerPage = %d, want 10 extracted from Link header", resp.PerPage)
	}

	nextOpts := resp.NextPageOptions()
	if nextOpts == nil || nextOpts.Page != 2 || nextOpts.PerPage != 10 {
		t.Fatalf("nextOpts invalid: %+v", nextOpts)
	}

	nextPath, err := AddOptions("items", nextOpts)
	if err != nil {
		t.Fatalf("AddOptions for page 2 failed: %v", err)
	}

	req2, err := client.NewRequest(http.MethodGet, nextPath, nil)
	if err != nil {
		t.Fatalf("NewRequest for page 2 failed: %v", err)
	}

	var page2 []item
	resp2, err := client.Do(ctx, req2, &page2)
	if err != nil {
		t.Fatalf("Second request failed: %v", err)
	}

	if resp2.NextPage != 0 {
		t.Errorf("Expected last page to have NextPage 0, got %d", resp2.NextPage)
	}

	if len(observedPerPages) != 2 {
		t.Fatalf("Expected 2 requests, got %d", len(observedPerPages))
	}
	if observedPerPages[0] != "" {
		t.Errorf("Request 1 sent per_page: %q, expected empty", observedPerPages[0])
	}
	if observedPerPages[1] != "10" {
		t.Errorf("Request 2 sent per_page: %q, expected 10 preserved from response options", observedPerPages[1])
	}
}

func TestClient_NextPage_Helper(t *testing.T) {
	type item struct {
		ID int `json:"id"`
	}

	var serverURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		pageStr := r.URL.Query().Get("page")
		switch pageStr {
		case "", "1":
			w.Header().Set("Link", fmt.Sprintf(`<%s/items?page=2>; rel="next"`, serverURL))
			json.NewEncoder(w).Encode([]item{{ID: 1}})
		case "2":
			json.NewEncoder(w).Encode([]item{{ID: 2}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	serverURL = server.URL

	client := NewClient(server.Client())
	parsedBaseURL, _ := url.Parse(server.URL + "/")
	client.BaseURL = parsedBaseURL

	req, _ := client.NewRequest(http.MethodGet, "items", nil)
	var items1 []item
	resp1, err := client.Do(context.Background(), req, &items1)
	if err != nil {
		t.Fatalf("client.Do failed: %v", err)
	}

	var items2 []item
	resp2, err := client.NextPage(context.Background(), resp1, &items2)
	if err != nil {
		t.Fatalf("client.NextPage failed: %v", err)
	}
	if resp2.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp2.StatusCode)
	}
	if len(items2) != 1 || items2[0].ID != 2 {
		t.Errorf("unexpected items2: %+v", items2)
	}
}
