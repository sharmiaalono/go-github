import math
import re
import urllib.parse
from dataclasses import dataclass, field
from typing import Dict


@dataclass
class ListOptions:
    """Lightweight analogue of go-github's ListOptions.

    A per_page value of 0 is treated as using the GitHub API default (typically 30)
    and is omitted when constructing query strings to mirror the omitempty behavior.
    """

    page: int = 0
    per_page: int = 0

    def effective_per_page(self, default: int = 30) -> int:
        return self.per_page if self.per_page else default


class Response:
    """Simple pagination response container."""

    def __init__(self) -> None:
        self.NextPage: int = 0
        self.PrevPage: int = 0
        self.FirstPage: int = 0
        self.LastPage: int = 0
        self.Links: Dict[str, str] = {}

    def __repr__(self) -> str:  # pragma: no cover - debugging helper
        return (
            f"Response(NextPage={self.NextPage}, PrevPage={self.PrevPage}, "
            f"FirstPage={self.FirstPage}, LastPage={self.LastPage}, Links={self.Links})"
        )


def _options_to_query(options) -> Dict[str, list]:
    """Convert supported options to a querystring-ready mapping."""
    if options is None:
        return {}

    if isinstance(options, dict):
        # Only filter out zero values for known pagination fields.
        return {
            k: [str(v)]
            for k, v in options.items()
            if v is not None and not (k in {"page", "per_page"} and v == 0)
        }

    q: Dict[str, list] = {}
    page = getattr(options, "page", getattr(options, "Page", 0))
    per_page = getattr(options, "per_page", getattr(options, "PerPage", 0))

    if page:
        q["page"] = [str(page)]
    if per_page:
        q["per_page"] = [str(per_page)]

    return q


def add_options(url: str, options) -> str:
    """Append option fields to a URL's querystring without dropping existing values.

    Mimics go-github's addOptions: when per_page is 0 it is omitted entirely to allow
    the server to apply the default page size while still including explicit page values.
    """

    parsed = urllib.parse.urlparse(url)
    base_qs = urllib.parse.parse_qs(parsed.query, keep_blank_values=True)
    opt_qs = _options_to_query(options)

    for key, values in opt_qs.items():
        for value in values:
            base_qs.setdefault(key, []).append(value)

    new_query = urllib.parse.urlencode(base_qs, doseq=True)
    return parsed._replace(query=new_query).geturl()


def parse_pagination(link_header: str) -> Response:
    """Parse an RFC 5988 Link header into pagination fields.

    The function is indifferent to whether a per_page parameter is present; it only
    extracts the page number from each relation, mirroring github.Response helpers.
    """

    resp = Response()
    if not link_header:
        return resp

    pattern = re.compile(r"<([^>]+)>;\s*rel=\"([^\"]+)\"")
    for part in (p.strip() for p in link_header.split(",") if p.strip()):
        match = pattern.match(part)
        if not match:
            continue
        url, rel = match.groups()
        resp.Links[rel] = url

        parsed = urllib.parse.urlparse(url)
        qs = urllib.parse.parse_qs(parsed.query)
        page_values = qs.get("page")
        try:
            page = int(page_values[0]) if page_values else 0
        except (TypeError, ValueError):
            page = 0

        if rel == "next":
            resp.NextPage = page
        elif rel == "prev":
            resp.PrevPage = page
        elif rel == "first":
            resp.FirstPage = page
        elif rel == "last":
            resp.LastPage = page

    return resp


def paginate_items(items, options: ListOptions, base_url: str) -> tuple:
    """Simulate a paginated API response for testing purposes.

    Uses the effective per_page when zero to ensure default sizing is respected and
    constructs a Link header that includes per_page and page parameters.
    """

    per_page = options.effective_per_page()
    page = options.page if options.page else 1
    total_pages = math.ceil(len(items) / per_page) if per_page else 0

    start = (page - 1) * per_page
    end = start + per_page
    current_items = items[start:end]

    links = []
    if page < total_pages:
        links.append(
            f'<{add_options(base_url, ListOptions(page=page + 1, per_page=per_page))}>; rel="next"'
        )
    if page > 1:
        links.append(
            f'<{add_options(base_url, ListOptions(page=page - 1, per_page=per_page))}>; rel="prev"'
        )
    if total_pages:
        links.append(
            f'<{add_options(base_url, ListOptions(page=1, per_page=per_page))}>; rel="first"'
        )
        links.append(
            f'<{add_options(base_url, ListOptions(page=total_pages, per_page=per_page))}>; rel="last"'
        )

    link_header = ", ".join(links)
    return current_items, link_header
