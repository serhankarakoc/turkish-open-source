package github

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

var ErrInvalidSearchQuery = errors.New("invalid search query")

type SearchPage struct {
	TotalCount        int
	IncompleteResults bool
	Items             int
}

func (c *Client) SearchRepositories(ctx context.Context, query string, maxPages, perPage int) ([]Repository, error) {
	if perPage <= 0 || perPage > 100 {
		perPage = 100
	}
	if maxPages <= 0 {
		maxPages = 1
	}
	var out []Repository
	_, err := c.search(ctx, "/search/repositories", query, maxPages, perPage, func(raw []byte) error {
		var page struct {
			Items []Repository `json:"items"`
		}
		if err := decodeJSON(raw, &page); err != nil {
			return err
		}
		out = append(out, page.Items...)
		return nil
	})
	return out, err
}

func (c *Client) SearchUsers(ctx context.Context, query string, maxPages, perPage int) ([]User, error) {
	if perPage <= 0 || perPage > 100 {
		perPage = 100
	}
	if maxPages <= 0 {
		maxPages = 1
	}
	var out []User
	_, err := c.search(ctx, "/search/users", query, maxPages, perPage, func(raw []byte) error {
		var page struct {
			Items []User `json:"items"`
		}
		if err := decodeJSON(raw, &page); err != nil {
			return err
		}
		out = append(out, page.Items...)
		return nil
	})
	return out, err
}

func (c *Client) search(ctx context.Context, path, query string, maxPages, perPage int, consume func([]byte) error) (SearchPage, error) {
	var meta SearchPage
	for page := 1; page <= maxPages; page++ {
		q := url.Values{}
		q.Set("q", query)
		q.Set("per_page", strconv.Itoa(perPage))
		q.Set("page", strconv.Itoa(page))
		body, status, err := c.do(ctx, "GET", path, q, false)
		if err != nil {
			if IsNotFound(err) {
				return meta, nil
			}
			var api APIError
			if asAPIError(err, &api) && (api.Status == 422 || api.Status == 400) {
				return meta, fmt.Errorf("%w: %q", ErrInvalidSearchQuery, query)
			}
			return meta, err
		}
		if status != 200 {
			return meta, fmt.Errorf("search %s: unexpected status %d", path, status)
		}
		var header struct {
			TotalCount        int  `json:"total_count"`
			IncompleteResults bool `json:"incomplete_results"`
		}
		if err := decodeJSON(body, &header); err != nil {
			return meta, err
		}
		meta.TotalCount = header.TotalCount
		meta.IncompleteResults = header.IncompleteResults
		if err := consume(body); err != nil {
			return meta, err
		}
		if page*perPage >= header.TotalCount {
			break
		}
	}
	return meta, nil
}

func ParseLinkNext(linkHeader string) string {
	if linkHeader == "" {
		return ""
	}
	parts := strings.Split(linkHeader, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if !strings.Contains(part, `rel="next"`) {
			continue
		}
		start := strings.Index(part, "<")
		end := strings.Index(part, ">")
		if start >= 0 && end > start {
			return part[start+1 : end]
		}
	}
	return ""
}

func asAPIError(err error, dest *APIError) bool {
	if err == nil {
		return false
	}
	e, ok := err.(APIError)
	if ok {
		*dest = e
		return true
	}
	return false
}
