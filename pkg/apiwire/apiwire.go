// Implements: REQ-021.
// Per: ADR-0075.
// Discipline: C-14.

// Package apiwire owns the wire contract shared by every PlatformKit JSON
// API party: the canonical list-query parameter names (with their legacy
// aliases) and the item/list response envelopes. Servers parse queries with
// ParseListQuery and serialize Item/List; clients (pk-client) encode the
// canonical parameters and decode the same envelopes. Keeping the vocabulary
// here — in the dependency-graph leaf — lets both sides of the wire bind to
// it without coupling to each other.
//
// Error bodies are deliberately NOT defined here. They are RFC 7807 problem
// documents owned by github.com/septagon-oss/problem, which is already the
// error contract elsewhere in the family. Duplicating a second error shape in
// this package would recreate exactly the drift this package exists to
// remove, and importing problem here would cost pk-shared its zero-dependency
// leaf status — which is what makes it safe for every party to depend on.
// Servers import problem directly where they write errors.
//
// ADR: ADR-0029 (file purpose declaration), ADR-0075 (canonical wire contract).
// Convention: C-14 (every Go file declares its purpose).
package apiwire

import (
	"net/url"
	"strconv"
)

// Canonical list-query parameter names. These are what pk-client encodes and
// what every server accepts first.
const (
	ParamPage     = "page"
	ParamPageSize = "page_size"
	ParamOffset   = "offset"
	ParamSort     = "sort"
	ParamOrder    = "order"
	ParamSearch   = "search"
)

// Legacy aliases accepted by ParseListQuery for compatibility with callers
// that predate the canonical names. Canonical parameters win when both are
// present.
const (
	ParamLimit = "limit" // alias for page_size
	ParamDesc  = "desc"  // alias for order=desc
	ParamQ     = "q"     // alias for search
)

// Order values carried by ParamOrder.
const (
	OrderAsc  = "asc"
	OrderDesc = "desc"
)

// ListQuery is the normalized form of a list request's query string. Zero
// values mean "unspecified"; policy (default and maximum page sizes) belongs
// to the server, not the wire vocabulary.
type ListQuery struct {
	Page     int
	PageSize int
	Offset   int
	Sort     string
	Desc     bool
	Search   string
}

// ParseListQuery normalizes a query string into a ListQuery, accepting the
// canonical parameters first and the legacy aliases (limit, desc, q) when the
// canonical form is absent. Invalid or negative integers parse as zero.
func ParseListQuery(values url.Values) ListQuery {
	q := ListQuery{
		Page:     nonNegativeInt(values.Get(ParamPage)),
		PageSize: nonNegativeInt(values.Get(ParamPageSize)),
		Offset:   nonNegativeInt(values.Get(ParamOffset)),
		Sort:     values.Get(ParamSort),
		Search:   values.Get(ParamSearch),
	}
	if q.PageSize == 0 {
		q.PageSize = nonNegativeInt(values.Get(ParamLimit))
	}
	if q.Search == "" {
		q.Search = values.Get(ParamQ)
	}
	switch values.Get(ParamOrder) {
	case OrderDesc:
		q.Desc = true
	case OrderAsc:
		q.Desc = false
	default:
		switch values.Get(ParamDesc) {
		case "1", "true", "TRUE", "True":
			q.Desc = true
		}
	}
	return q
}

// EffectiveOffset resolves the two pagination styles into one row offset: an
// explicit offset wins, otherwise page/page_size derive it, otherwise zero.
func (q ListQuery) EffectiveOffset() int {
	if q.Offset > 0 {
		return q.Offset
	}
	if q.Page > 1 && q.PageSize > 0 {
		return (q.Page - 1) * q.PageSize
	}
	return 0
}

// EffectiveLimit is the requested page size (zero when unspecified — the
// server chooses its own default).
func (q ListQuery) EffectiveLimit() int {
	return q.PageSize
}

// PageMetadata builds the pagination metadata a server can assert from the
// query alone. Total counts are left zero unless the caller sets them —
// stores that do not count matching rows simply omit them.
func (q ListQuery) PageMetadata() *ListMetadata {
	page := q.Page
	if page == 0 {
		if q.PageSize > 0 && q.Offset > 0 {
			page = q.Offset/q.PageSize + 1
		} else {
			page = 1
		}
	}
	return &ListMetadata{Page: page, PageSize: q.PageSize}
}

// Item is the envelope for a single entity.
type Item[T any] struct {
	Data     T              `json:"data"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// List is the envelope for a page of entities.
type List[T any] struct {
	Data     []T           `json:"data"`
	Metadata *ListMetadata `json:"metadata,omitempty"`
}

// ListMetadata describes the pagination state of a List response.
type ListMetadata struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalCount int64 `json:"total_count"`
	TotalPages int   `json:"total_pages"`
}

func nonNegativeInt(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return 0
	}
	return n
}
