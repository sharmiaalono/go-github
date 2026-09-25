# Pagination Defaults

The client library normalizes pagination options to provide consistent and safe defaults across all endpoints.

- **PerPage**: If omitted (zero value) or explicitly set to `0`, the library will use `DefaultPerPage` (currently `30`).
- **Page**: If omitted or set to `0`, the library defaults to the first page (`1`).
- Values exceeding the limits are capped (`PerPage` is capped at `MaxPerPage` = 100).

This behavior is implemented by `normalizeListOptions` and is applied automatically by `BuildListURL` and the `SetDefaults` helper on `ListOptions`.
