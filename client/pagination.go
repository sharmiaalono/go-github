package client

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/google/go-querystring/query"
)

type ListOptions struct {
	Page    int `url:"page,omitempty"`
	PerPage int `url:"per_page,omitempty"`
}

type Response struct {
	*http.Response
	NextPage  int
	PrevPage  int
	FirstPage int
	LastPage  int
}

func (r *Response) NextPageNum() int  { return r.NextPage }
func (r *Response) PrevPageNum() int  { return r.PrevPage }
func (r *Response) FirstPageNum() int { return r.FirstPage }
func (r *Response) LastPageNum() int  { return r.LastPage }

func addOptions(s string, opt interface{}) (string, error) {
	v, err := query.Values(opt)
	if err != nil {
		return s, err
	}

	u, err := url.Parse(s)
	if err != nil {
		return s, err
	}

	qs := u.Query()
	for k, values := range v {
		for _, value := range values {
			if k == "per_page" && value == "0" {
				continue
			}
			qs.Add(k, value)
		}
	}

	u.RawQuery = qs.Encode()
	return u.String(), nil
}

func parsePagination(resp *http.Response) (*Response, error) {
	r := &Response{Response: resp}
	linkHeader := resp.Header.Get("Link")
	if linkHeader == "" {
		return r, nil
	}

	links := strings.Split(linkHeader, ",")
	for _, link := range links {
		fields := strings.Split(strings.TrimSpace(link), ";")
		if len(fields) < 2 {
			continue
		}

		rawURL := strings.Trim(fields[0], "<>")
		u, err := url.Parse(rawURL)
		if err != nil {
			continue
		}

		q := u.Query()
		pageStr := q.Get("page")
		if pageStr == "" {
			continue
		}

		page, err := strconv.Atoi(pageStr)
		if err != nil {
			continue
		}

		var rel string
		for _, field := range fields[1:] {
			field = strings.TrimSpace(field)
			if strings.HasPrefix(field, "rel=") {
				rel = strings.Trim(field[4:], `"'`)
				break
			}
		}

		switch rel {
		case "next":
			r.NextPage = page
		case "prev":
			r.PrevPage = page
		case "first":
			r.FirstPage = page
		case "last":
			r.LastPage = page
		}
	}

	return r, nil
}
