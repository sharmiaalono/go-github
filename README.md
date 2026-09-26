# go-github

A Go client library for interacting with APIs.

## Pagination

Pagination is supported via `ListOptions`:

```go
type ListOptions struct {
    Page    int `url:"page,omitempty"`
    PerPage int `url:"per_page,omitempty"`
}
```

When `PerPage` is set to `0`, the `per_page` query parameter is safely omitted from the request URL, allowing the server to use its default page size.

### Example

```go
opts := &client.ListOptions{Page: 1, PerPage: 0}
resp, err := client.ListSomething(ctx, opts)
if err != nil {
    // handle error
}

if resp.NextPage() != 0 {
    // fetch next page
}
```
