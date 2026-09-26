# go-github

## Pagination

This library provides built-in pagination support for GitHub API list operations.

### ListOptions

Use `ListOptions` to control pagination:

```go
type ListOptions struct {
    Page    int `url:"page,omitempty"`
    PerPage int `url:"per_page,omitempty"`
}
```

- Set `PerPage: 0` (or omit it) to use the GitHub default page size. The `per_page` query parameter will be safely omitted from the request.
- Set `PerPage` to a positive value to request a specific page size.

### Response

API methods return a `*client.Response` with pagination metadata parsed from the `Link` header:

```go
type Response struct {
    *http.Response
    NextPage  int
    PrevPage  int
    FirstPage int
    LastPage  int
}
```

### Example: Multi-page fetching

```go
client := client.NewClient(nil)

var allIssues []client.Issue
opts := &client.ListOptions{Page: 1, PerPage: 0}

for {
    issues, resp, err := client.ListIssues("owner", "repo", opts)
    if err != nil {
        log.Fatal(err)
    }
    allIssues = append(allIssues, issues...)

    if resp.NextPage == 0 {
        break
    }
    opts.Page = resp.NextPage
}
```

### Behavior

- `PerPage: 0` is never sent as a query parameter.
- Pagination helpers (`NextPage`, `PrevPage`, `FirstPage`, `LastPage`) correctly return page numbers even when the initial request omitted `per_page`.
- URL construction preserves existing query parameters and adds pagination parameters correctly.
