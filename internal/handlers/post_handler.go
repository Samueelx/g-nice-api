package handlers

import (
	"errors"
	"log"
	"strconv"

	"github.com/Samueelx/g-nice-api/internal/services"
	"github.com/gin-gonic/gin"
)

// PostHandler handles HTTP requests for post operations.
type PostHandler struct {
	postSvc services.PostService
}

// NewPostHandler constructs a PostHandler.
func NewPostHandler(postSvc services.PostService) *PostHandler {
	return &PostHandler{postSvc: postSvc}
}

// CreatePost godoc
// POST /api/v1/posts
// Body: { content, media_url?, media_type?, is_public? }
func (h *PostHandler) CreatePost(c *gin.Context) {
	userID, ok := extractUserID(c)
	if !ok {
		return
	}

	var req services.CreatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	post, err := h.postSvc.CreatePost(userID, &req)
	if err != nil {
		log.Printf("CreatePost error: %v", err)
		InternalError(c)
		return
	}

	Created(c, post)
}

// GetPost godoc
// GET /api/v1/posts/:id
func (h *PostHandler) GetPost(c *gin.Context) {
	postID, ok := parsePostID(c)
	if !ok {
		return
	}

	post, err := h.postSvc.GetPost(postID)
	if err != nil {
		if errors.Is(err, services.ErrPostNotFound) {
			NotFound(c, "post not found")
			return
		}
		log.Printf("GetPost error: %v", err)
		InternalError(c)
		return
	}

	OK(c, post)
}

// UpdatePost godoc
// PATCH /api/v1/posts/:id
// Body: { content?, is_public? }
func (h *PostHandler) UpdatePost(c *gin.Context) {
	userID, ok := extractUserID(c)
	if !ok {
		return
	}
	postID, ok := parsePostID(c)
	if !ok {
		return
	}

	var req services.UpdatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	post, err := h.postSvc.UpdatePost(userID, postID, &req)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrPostNotFound):
			NotFound(c, "post not found")
		case errors.Is(err, services.ErrForbidden):
			Forbidden(c, "you can only edit your own posts")
		default:
			log.Printf("UpdatePost error: %v", err)
			InternalError(c)
		}
		return
	}

	OK(c, post)
}

// DeletePost godoc
// DELETE /api/v1/posts/:id
func (h *PostHandler) DeletePost(c *gin.Context) {
	userID, ok := extractUserID(c)
	if !ok {
		return
	}
	postID, ok := parsePostID(c)
	if !ok {
		return
	}

	err := h.postSvc.DeletePost(userID, postID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrPostNotFound):
			NotFound(c, "post not found")
		case errors.Is(err, services.ErrForbidden):
			Forbidden(c, "you can only delete your own posts")
		default:
			log.Printf("DeletePost error: %v", err)
			InternalError(c)
		}
		return
	}

	ok204(c)
}

// ListFeed godoc
// GET /api/v1/posts?page=1&page_size=20
// Returns a paginated feed of all public posts, newest first.
func (h *PostHandler) ListFeed(c *gin.Context) {
	page, pageSize := parsePagination(c)

	result, err := h.postSvc.ListFeed(page, pageSize)
	if err != nil {
		log.Printf("ListFeed error: %v", err)
		InternalError(c)
		return
	}

	OK(c, result)
}

// ListUserPosts godoc
// GET /api/v1/users/:username/posts?page=1&page_size=20
// Returns a paginated list of public posts for the given user.
func (h *PostHandler) ListUserPosts(c *gin.Context) {
	username := c.Param("username")
	if username == "" {
		BadRequest(c, "username is required")
		return
	}

	page, pageSize := parsePagination(c)

	result, err := h.postSvc.ListUserPosts(username, page, pageSize)
	if err != nil {
		if errors.Is(err, services.ErrProfileNotFound) {
			NotFound(c, "user not found")
			return
		}
		log.Printf("ListUserPosts error: %v", err)
		InternalError(c)
		return
	}

	OK(c, result)
}

// ── Private helpers ───────────────────────────────────────────────────────────

func parsePostID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		BadRequest(c, "invalid post id")
		return 0, false
	}
	return uint(id), true
}
