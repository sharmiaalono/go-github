# go-github

This repository contains a focused `github` client package for paginated list
requests. It preserves existing query parameters, omits `per_page` when
`ListOptions.PerPage` is zero, parses RFC 5988 `Link` headers, and exposes the
server-selected page size so follow-up requests remain consistent.

The mock integration test follows the `Link: rel="next"` URL semantics across
multiple pages while retaining endpoint filters such as `state` and `labels`.
