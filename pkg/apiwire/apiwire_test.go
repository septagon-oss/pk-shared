// Implements: REQ-021.
// Per: ADR-0075.
// Discipline: C-14.

// apiwire_test.go pins the dialect-normalization rules of ParseListQuery,
// the offset arithmetic, and the exact JSON shapes of the envelopes.
//
// ADR: ADR-0029 (file purpose declaration).
// Convention: C-14 (every Go file declares its purpose).
package apiwire

import (
	"encoding/json"
	"net/url"
	"testing"
)

func TestParseListQueryDialects(t *testing.T) {
	cases := []struct {
		name  string
		query string
		want  ListQuery
	}{
		{"empty", "", ListQuery{}},
		{"canonical", "page=2&page_size=25&sort=name&order=desc&search=jo",
			ListQuery{Page: 2, PageSize: 25, Sort: "name", Desc: true, Search: "jo"}},
		{"legacy", "limit=10&offset=30&sort=name&desc=true&q=jo",
			ListQuery{PageSize: 10, Offset: 30, Sort: "name", Desc: true, Search: "jo"}},
		{"canonical wins over alias", "page_size=25&limit=99&search=a&q=b&order=asc&desc=true",
			ListQuery{PageSize: 25, Search: "a"}},
		{"invalid ints parse as zero", "page=x&page_size=-5&offset=1.5", ListQuery{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			values, err := url.ParseQuery(tc.query)
			if err != nil {
				t.Fatalf("parse query: %v", err)
			}
			if got := ParseListQuery(values); got != tc.want {
				t.Fatalf("got %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestEffectiveOffset(t *testing.T) {
	cases := []struct {
		name string
		q    ListQuery
		want int
	}{
		{"unspecified", ListQuery{}, 0},
		{"explicit offset wins", ListQuery{Offset: 40, Page: 3, PageSize: 10}, 40},
		{"derived from page", ListQuery{Page: 3, PageSize: 10}, 20},
		{"page one is zero", ListQuery{Page: 1, PageSize: 10}, 0},
		{"page without size", ListQuery{Page: 3}, 0},
	}
	for _, tc := range cases {
		if got := tc.q.EffectiveOffset(); got != tc.want {
			t.Errorf("%s: got %d, want %d", tc.name, got, tc.want)
		}
	}
}

func TestPageMetadata(t *testing.T) {
	if m := (ListQuery{Page: 2, PageSize: 25}).PageMetadata(); m.Page != 2 || m.PageSize != 25 {
		t.Fatalf("got %+v", m)
	}
	if m := (ListQuery{Offset: 50, PageSize: 25}).PageMetadata(); m.Page != 3 {
		t.Fatalf("offset-derived page: got %+v", m)
	}
	if m := (ListQuery{}).PageMetadata(); m.Page != 1 {
		t.Fatalf("default page: got %+v", m)
	}
}

func TestEnvelopeJSONShapes(t *testing.T) {
	type entity struct {
		ID string `json:"id"`
	}
	item, err := json.Marshal(Item[entity]{Data: entity{ID: "e1"}})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(item), `{"data":{"id":"e1"}}`; got != want {
		t.Fatalf("item envelope: got %s, want %s", got, want)
	}
	list, err := json.Marshal(List[entity]{
		Data:     []entity{{ID: "e1"}},
		Metadata: &ListMetadata{Page: 1, PageSize: 25},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"data":[{"id":"e1"}],"metadata":{"page":1,"page_size":25,"total_count":0,"total_pages":0}}`
	if got := string(list); got != want {
		t.Fatalf("list envelope: got %s, want %s", got, want)
	}
	// Error bodies are RFC 7807 documents owned by septagon-oss/problem, not
	// this package — see the package doc.

	// Empty list must serialize as [], never null — clients range over it.
	empty, err := json.Marshal(List[entity]{Data: []entity{}})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(empty), `{"data":[]}`; got != want {
		t.Fatalf("empty list envelope: got %s, want %s", got, want)
	}
}
