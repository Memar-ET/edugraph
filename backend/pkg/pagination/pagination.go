// Package pagination provides shared query-param parsing and response
// metadata for paginated list endpoints.
package pagination

import (
	"net/http"
	"strconv"
)

const (
	defaultLimit = 20
	maxLimit     = 100
)

type Params struct {
	Page  int
	Limit int
}

func FromRequest(r *http.Request) Params {
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}
	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil || limit < 1 || limit > maxLimit {
		limit = defaultLimit
	}
	return Params{Page: page, Limit: limit}
}

func (p Params) Offset() int {
	return (p.Page - 1) * p.Limit
}

type Meta struct {
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
	Total int64 `json:"total"`
}

func NewMeta(p Params, total int64) Meta {
	return Meta{Page: p.Page, Limit: p.Limit, Total: total}
}
