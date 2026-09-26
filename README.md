# go-github

## Pagination

Pagination helpers allow fetching large collections easily using `ListOptions`.

```go
opts := &client.ListOptions{Page: 1, PerPage: 0}
```

When `PerPage` is set to `0`, the `per_page` query parameter is safely omitted from requests, leaving it up to the server default.
