# go-github

## Pagination

Pagination in `go-github` is supported using `ListOptions`:

```go
type ListOptions struct {
    Page    int `url:"page,omitempty"`
    PerPage int `url:"per_page,omitempty"`
}
```

When `PerPage` is set to `0`, the `per_page` query parameter is safely omitted from requests, allowing the API to use its default page size.

### Example: Multi-page Fetching

```go
opts := &client.ListOptions{Page: 1, PerPage: 0}
for {
    items, resp, err := client.List(ctx, "https://api.example.com/items", opts)
    if err != nil {
        // handle error
    }
    // process items...
    if resp.NextPage == 0 {
        break
    }
    opts.Page = resp.NextPage
}
```
