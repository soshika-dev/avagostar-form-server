package utils

import "math"

type Pagination struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

func NewPagination(page, perPage int, total int64) Pagination {
	if page < 1 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 10
	}
	totalPages := int(math.Ceil(float64(total) / float64(perPage)))
	return Pagination{
		Page:       page,
		PerPage:    perPage,
		Total:      int(total),
		TotalPages: totalPages,
	}
}
