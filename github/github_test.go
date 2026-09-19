package github

import "testing"

func TestParsePaginationWithServerDefault(t *testing.T) {
	h := `<https://api.example.test/items?page=2&per_page=30>; rel="next", <https://api.example.test/items?page=1&per_page=30>; rel="first", <https://api.example.test/items?page=4&per_page=30>; rel="last"`
	r := ParsePagination(h)
	if r.NextPage != 2 || r.FirstPage != 1 || r.LastPage != 4 || r.PerPage != 30 {
		t.Fatalf("unexpected pagination: %#v", r)
	}
	got := PageOptions(ListOptions{Page: 1, PerPage: 0}, r.NextPage, r)
	if got.Page != 2 || got.PerPage != 30 {
		t.Fatalf("lost link options: %#v", got)
	}
}

func TestParsePaginationIgnoresMalformedLinks(t *testing.T) {
	r := ParsePagination(`garbage, <not a url>; rel="next", <https://x.test/items?page=0>; rel="last"`)
	if r != (Response{}) {
		t.Fatalf("malformed header should be ignored: %#v", r)
	}
}

func TestPageOptionsKeepsExplicitPageSize(t *testing.T) {
	got := PageOptions(ListOptions{Page: 3, PerPage: 50}, 4, Response{PerPage: 30})
	if got.Page != 4 || got.PerPage != 50 {
		t.Fatalf("explicit page size changed: %#v", got)
	}
}
