package pagination

import "testing"

func TestNewPaginationParamsSanitizesValues(t *testing.T) {
	tests := []struct {
		name       string
		page       int
		limit      int
		wantPage   int
		wantLimit  int
		wantOffset int
	}{
		{name: "defaults invalid values", page: 0, limit: 0, wantPage: 1, wantLimit: 10, wantOffset: 0},
		{name: "keeps valid values", page: 3, limit: 25, wantPage: 3, wantLimit: 25, wantOffset: 50},
		{name: "caps maximum limit", page: 2, limit: 101, wantPage: 2, wantLimit: 100, wantOffset: 100},
		{name: "normalizes negative values", page: -1, limit: -5, wantPage: 1, wantLimit: 10, wantOffset: 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			params := NewPaginationParams(test.page, test.limit)

			if params.Page != test.wantPage || params.Limit != test.wantLimit {
				t.Fatalf("got page=%d limit=%d, want page=%d limit=%d",
					params.Page, params.Limit, test.wantPage, test.wantLimit)
			}
			if params.Offset() != test.wantOffset {
				t.Fatalf("got offset=%d, want %d", params.Offset(), test.wantOffset)
			}
		})
	}
}
