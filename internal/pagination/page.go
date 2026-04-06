package pagination

import "math"

// Page is a generic paginated result container.
// T can be any type — models.Post, models.Comment, etc.
type Page[T any] struct {
	Data       []T   `json:"data"`
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalPages int   `json:"total_pages"`
	HasNext    bool  `json:"has_next"`
	HasPrev    bool  `json:"has_prev"`
}

// New constructs a Page from a data slice, the total count, and the pagination params.
func New[T any](data []T, total int64, page, pageSize int) *Page[T] {
	if data == nil {
		data = []T{} // always return an array, never null in JSON
	}

	totalPages := 1
	if pageSize > 0 && total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(pageSize)))
	}

	return &Page[T]{
		Data:       data,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}
}

// Offset returns the SQL OFFSET value for the given page and page size.
func Offset(page, pageSize int) int {
	if page < 1 {
		page = 1
	}
	return (page - 1) * pageSize
}
