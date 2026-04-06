package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ── Auth helpers ──────────────────────────────────────────────────────────────

// extractUserID reads the userID injected by AuthRequired middleware.
// Writes a 401 and returns false if the value is missing or malformed.
func extractUserID(c *gin.Context) (uint, bool) {
	v, exists := c.Get("userID")
	if !exists {
		Unauthorized(c, "not authenticated")
		return 0, false
	}
	id, ok := v.(uint)
	if !ok || id == 0 {
		Unauthorized(c, "invalid token claims")
		return 0, false
	}
	return id, true
}

// ── Pagination helpers ────────────────────────────────────────────────────────

const (
	defaultPageSize = 20
	maxPageSize     = 100
)

// parsePagination reads ?page and ?page_size from the query string with sane
// defaults and upper bounds. Returns (page, pageSize) both >= 1.
func parsePagination(c *gin.Context) (page, pageSize int) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err = strconv.Atoi(c.DefaultQuery("page_size", strconv.Itoa(defaultPageSize)))
	if err != nil || pageSize < 1 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	return page, pageSize
}

// ok204 sends a 204 No Content on successful deletion.
func ok204(c *gin.Context) {
	c.Status(http.StatusNoContent)
}
