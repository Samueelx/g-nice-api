package handlers

import (
	"log"

	"github.com/Samueelx/g-nice-api/internal/services"
	"github.com/gin-gonic/gin"
)

// SearchHandler handles HTTP requests for the unified search endpoint.
type SearchHandler struct {
	searchSvc services.SearchService
}

// NewSearchHandler constructs a SearchHandler.
func NewSearchHandler(searchSvc services.SearchService) *SearchHandler {
	return &SearchHandler{searchSvc: searchSvc}
}

// Search godoc
//
//	GET /api/v1/search?q=<term>&type=all|users|posts&page=1&page_size=20
//
// Returns a unified search result containing users and/or posts that match the query.
// The `type` parameter controls which resources are searched (default: all).
// This endpoint is public and does not require authentication.
func (h *SearchHandler) Search(c *gin.Context) {
	q := c.Query("q")
	if q == "" {
		BadRequest(c, "query parameter 'q' is required")
		return
	}

	searchType := services.SearchType(c.DefaultQuery("type", string(services.SearchAll)))
	page, pageSize := parsePagination(c)

	result, err := h.searchSvc.Search(q, searchType, page, pageSize)
	if err != nil {
		log.Printf("Search error: %v", err)
		InternalError(c)
		return
	}

	OK(c, result)
}
