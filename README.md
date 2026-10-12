# go-github

Small GitHub API pagination helpers used for BountyHub issue [#5](https://github.com/sharmiaalono/go-github/issues/5).

## Behavior (`ListOptions.PerPage == 0`)

- `per_page` is **omitted** from the query string (`url:",omitempty"`), so GitHub applies its default page size.
- `Response` still parses `page` from RFC 5988 `Link` relations (`next` / `prev` / `first` / `last`) whether or not `per_page` is present in those URLs.
- `ListOptions.WithPage` keeps the current `PerPage` when walking pages.

## Test

```bash
go test ./...
```
