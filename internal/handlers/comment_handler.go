package handlers

import (
	"errors"
	"log"
	"strconv"

	"github.com/Samueelx/g-nice-api/internal/services"
	"github.com/gin-gonic/gin"
)

// CommentHandler handles HTTP requests for comment operations.
type CommentHandler struct {
	commentSvc services.CommentService
}

// NewCommentHandler constructs a CommentHandler.
func NewCommentHandler(commentSvc services.CommentService) *CommentHandler {
	return &CommentHandler{commentSvc: commentSvc}
}

// CreateComment godoc
// POST /api/v1/posts/:id/comments
// Body: { content }
func (h *CommentHandler) CreateComment(c *gin.Context) {
	userID, ok := extractUserID(c)
	if !ok {
		return
	}
	postID, ok := parsePostID(c)
	if !ok {
		return
	}

	var req services.CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	comment, err := h.commentSvc.CreateComment(userID, postID, &req)
	if err != nil {
		if errors.Is(err, services.ErrPostNotFound) {
			NotFound(c, "post not found")
			return
		}
		log.Printf("CreateComment error: %v", err)
		InternalError(c)
		return
	}

	Created(c, comment)
}

// CreateReply godoc
// POST /api/v1/posts/:id/comments/:cid/replies
// Body: { content }
func (h *CommentHandler) CreateReply(c *gin.Context) {
	userID, ok := extractUserID(c)
	if !ok {
		return
	}
	postID, ok := parsePostID(c)
	if !ok {
		return
	}
	commentID, ok := parseCommentID(c)
	if !ok {
		return
	}

	var req services.CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	reply, err := h.commentSvc.CreateReply(userID, postID, commentID, &req)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrPostNotFound):
			NotFound(c, "post not found")
		case errors.Is(err, services.ErrCommentNotFound):
			NotFound(c, "comment not found")
		default:
			log.Printf("CreateReply error: %v", err)
			InternalError(c)
		}
		return
	}

	Created(c, reply)
}

// ListComments godoc
// GET /api/v1/posts/:id/comments?page=1&page_size=20
// Returns top-level comments for a post (paginated).
func (h *CommentHandler) ListComments(c *gin.Context) {
	postID, ok := parsePostID(c)
	if !ok {
		return
	}

	page, pageSize := parsePagination(c)

	result, err := h.commentSvc.ListComments(postID, page, pageSize)
	if err != nil {
		log.Printf("ListComments error: %v", err)
		InternalError(c)
		return
	}

	OK(c, result)
}

// ListReplies godoc
// GET /api/v1/comments/:cid/replies?page=1&page_size=20
// Returns threaded replies for a comment.
func (h *CommentHandler) ListReplies(c *gin.Context) {
	commentID, ok := parseCommentID(c)
	if !ok {
		return
	}

	page, pageSize := parsePagination(c)

	result, err := h.commentSvc.ListReplies(commentID, page, pageSize)
	if err != nil {
		if errors.Is(err, services.ErrCommentNotFound) {
			NotFound(c, "comment not found")
			return
		}
		log.Printf("ListReplies error: %v", err)
		InternalError(c)
		return
	}

	OK(c, result)
}

// UpdateComment godoc
// PATCH /api/v1/comments/:cid
// Body: { content }
func (h *CommentHandler) UpdateComment(c *gin.Context) {
	userID, ok := extractUserID(c)
	if !ok {
		return
	}
	commentID, ok := parseCommentID(c)
	if !ok {
		return
	}

	var req services.UpdateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	comment, err := h.commentSvc.UpdateComment(userID, commentID, &req)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrCommentNotFound):
			NotFound(c, "comment not found")
		case errors.Is(err, services.ErrForbidden):
			Forbidden(c, "you can only edit your own comments")
		default:
			log.Printf("UpdateComment error: %v", err)
			InternalError(c)
		}
		return
	}

	OK(c, comment)
}

// DeleteComment godoc
// DELETE /api/v1/comments/:cid
func (h *CommentHandler) DeleteComment(c *gin.Context) {
	userID, ok := extractUserID(c)
	if !ok {
		return
	}
	commentID, ok := parseCommentID(c)
	if !ok {
		return
	}

	err := h.commentSvc.DeleteComment(userID, commentID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrCommentNotFound):
			NotFound(c, "comment not found")
		case errors.Is(err, services.ErrForbidden):
			Forbidden(c, "you can only delete your own comments")
		default:
			log.Printf("DeleteComment error: %v", err)
			InternalError(c)
		}
		return
	}

	ok204(c)
}

// ── Private helper ────────────────────────────────────────────────────────────

func parseCommentID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("cid"), 10, 64)
	if err != nil || id == 0 {
		BadRequest(c, "invalid comment id")
		return 0, false
	}
	return uint(id), true
}
