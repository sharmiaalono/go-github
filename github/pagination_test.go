package github

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// helper to create a response with a Link header.
func newResponse(link string) *http.Response {
	rec := httptest.NewRecorder()
	rec.Header().Set("Link", link)
	return rec.Result()
}

func TestParsePagination_WithPerPageZero(t *testing.T) {
	// Simulate a response where the original request omitted per_page (i.e., PerPage=0).
	// The Link header also omits per_page.
	link := `<https://api.github.com/repos/octocat/Hello-World/issues?page=2>; rel="next", ` +
		`<https://api.github.com/repos/octocat/Hello-World/issues?page=5>; rel="last", ` +
		`<https://api.github.com/repos/octocat/Hello-World/issues?page=1>; rel="first", ` +
		`<https://api.github.com/repos/octocat/Hello-World/issues?page=1>; rel="prev"`

	resp := newResponse(link)

	parsed, err := parsePagination(resp)
	if err != nil {
		t.Fatalf("parsePagination returned error: %v", err)
	}
	if parsed.NextPage != 2 {
		t.Errorf("expected NextPage=2, got %d", parsed.NextPage)
	}
	if parsed.LastPage != 5 {
		t.Errorf("expected LastPage=5, got %d", parsed.LastPage)
	}
	if parsed.FirstPage != 1 {
		t.Errorf("expected FirstPage=1, got %d", parsed.FirstPage)
	}
	if parsed.PrevPage != 1 {
		t.Errorf("expected PrevPage=1, got %d", parsed.PrevPage)
	}
}

func TestAddOptions_PerPageZeroOmitted(t *testing.T) {
	base := "https://api.github.com/users"
	opts := &ListOptions{Page: 2, PerPage: 0}
	u, err := addOptions(base, opts)
	if err != nil {
		t.Fatalf("addOptions error: %v", err)
	}
	// per_page should not appear in the query string.
	if strings.Contains(u, "per_page") {
		t.Errorf("expected per_page to be omitted, got URL %s", u)
	}
	if !strings.Contains(u, "page=2") {
		t.Errorf("expected page=2 to be present, got URL %s", u)
	}
}
