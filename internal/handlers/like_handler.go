package handlers

import (
	"errors"
	"log"

	"github.com/Samueelx/g-nice-api/internal/services"
	"github.com/gin-gonic/gin"
)

// LikeHandler handles HTTP requests for like/unlike toggle operations.
type LikeHandler struct {
	likeSvc services.LikeService
}

// NewLikeHandler constructs a LikeHandler.
func NewLikeHandler(likeSvc services.LikeService) *LikeHandler {
	return &LikeHandler{likeSvc: likeSvc}
}

// TogglePostLike godoc
// POST /api/v1/posts/:id/like
// Likes the post if not already liked; unlikes it if it was.
// Response: { liked: bool, likes_count: int }
func (h *LikeHandler) TogglePostLike(c *gin.Context) {
	userID, ok := extractUserID(c)
	if !ok {
		return
	}
	postID, ok := parsePostID(c)
	if !ok {
		return
	}

	result, err := h.likeSvc.TogglePostLike(userID, postID)
	if err != nil {
		if errors.Is(err, services.ErrLikeTargetNotFound) {
			NotFound(c, "post not found")
			return
		}
		log.Printf("TogglePostLike error: %v", err)
		InternalError(c)
		return
	}

	OK(c, result)
}

// ToggleCommentLike godoc
// POST /api/v1/comments/:cid/like
// Likes the comment if not already liked; unlikes it if it was.
// Response: { liked: bool, likes_count: int }
func (h *LikeHandler) ToggleCommentLike(c *gin.Context) {
	userID, ok := extractUserID(c)
	if !ok {
		return
	}
	commentID, ok := parseCommentID(c)
	if !ok {
		return
	}

	result, err := h.likeSvc.ToggleCommentLike(userID, commentID)
	if err != nil {
		if errors.Is(err, services.ErrLikeTargetNotFound) {
			NotFound(c, "comment not found")
			return
		}
		log.Printf("ToggleCommentLike error: %v", err)
		InternalError(c)
		return
	}

	OK(c, result)
}
