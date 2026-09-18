// Response is a GitHub API response. This wraps the standard http.Response returned from GitHub.
type Response struct {
	*http.Response

	// NextPage and LastPage are the page numbers for the next and last page in the paginated result set.
	// These values are parsed from the Link response header. If no Link header is present, these values will be 0.
	NextPage int
	PrevPage int
	FirstPage int
	LastPage int
}

// parseLink parses the GitHub API pagination link header and extracts the page numbers for the next, previous, first, and last pages.
func parseLink(link string) (nextPage, prevPage, firstPage, lastPage int) {
	links := strings.Split(link, ",")
	for _, l := range links {
		parts := strings.Split(l, ";")
		if len(parts) != 2 {
			continue
		}
		urlPart := strings.TrimSpace(parts[0])
		relPart := strings.TrimSpace(parts[1])
		if relPart != `rel="next"` && relPart != `rel="prev"` && relPart != `rel="first"` && relPart != `rel="last"` {
			continue
		}
		url, err := url.Parse(strings.Trim(urlPart, "<>"))
		if err != nil {
			continue
		}
		page := url.Query().Get("page")
		if page == "" {
			continue
		}
		pageNum, err := strconv.Atoi(page)
		if err != nil {
			continue
		}
		switch relPart {
		case `rel="next"`:
			nextPage = pageNum
		case `rel="prev"`:
			prevPage = pageNum
		case `rel="first"`:
			firstPage = pageNum
		case `rel="last"`:
			lastPage = pageNum
		}
	}
	return nextPage, prevPage, firstPage, lastPage
}

// NewResponse creates a new Response for the provided http.Response.
func NewResponse(r *http.Response) *Response {
	response := &Response{Response: r}
	if link := r.Header.Get("Link"); link != "" {
		response.NextPage, response.PrevPage, response.FirstPage, response.LastPage = parseLink(link)
	}
	return response
}
