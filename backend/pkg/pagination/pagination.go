package pagination

type PaginationParams struct {
	Page  int
	Limit int
}

func NewPaginationParams(page, limit int) PaginationParams {
	p := PaginationParams{
		Page:  page,
		Limit: limit,
	}
	p.Sanitize()
	return p
}

func (p *PaginationParams) Sanitize() {
	minPageNumber := 1
	defaultLimitValue := 10
	maxLimit := 100
	if p.Page <= 0 {
		p.Page = minPageNumber
	}

	if p.Limit <= 0 {
		p.Limit = defaultLimitValue
	} else if p.Limit > maxLimit {
		p.Limit = maxLimit
	}
}

func (p PaginationParams) Offset() int {
	return (p.Page - 1) * p.Limit
}
