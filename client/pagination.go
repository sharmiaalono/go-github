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
	NextPage   int
	PrevPage   int
	FirstPage  int
	LastPage   int
}

func (r *Response) NextPageNum() int  { return r.NextPage }
func (r *Response) PrevPageNum() int  { return r.PrevPage }
func (r *Response) FirstPageNum() int { return r.FirstPage }
func (r *Response) LastPageNum() int  { return r.LastPage }

func addOptions(s string, opt interface{}) (string, error) {
	u, err := url.Parse(s)
	if err != nil {
		return s, err
	}

	v, err := query.Values(opt)
	if err != nil {
		return s, err
	}

	if v.Get("per_page") == "0" {
		v.Del("per_page")
	}

	qs := u.Query()
	for k, values := range v {
		for _, value := range values {
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
		items := strings.Split(strings.TrimSpace(link), ";")
		if len(items) < 2 {
			continue
		}

		rawURL := strings.Trim(items[0], "<>")
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

		for _, item := range items[1:] {
			item = strings.TrimSpace(item)
			if strings.HasPrefix(item, "rel=") {
				rel := strings.Trim(item[4:], `"'`)
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
		}
	}

	return r, nil
}
