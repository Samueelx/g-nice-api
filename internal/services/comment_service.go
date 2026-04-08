package services

import (
	"errors"

	"github.com/Samueelx/g-nice-api/internal/models"
	"github.com/Samueelx/g-nice-api/internal/pagination"
	"github.com/Samueelx/g-nice-api/internal/repository"
)

// ── DTOs ─────────────────────────────────────────────────────────────────────

// CreateCommentRequest is the payload for POST /posts/:id/comments
// and POST /posts/:id/comments/:cid/replies.
type CreateCommentRequest struct {
	Content string `json:"content" binding:"required,min=1,max=2000"`
}

// UpdateCommentRequest is the payload for PATCH /comments/:cid.
type UpdateCommentRequest struct {
	Content string `json:"content" binding:"required,min=1,max=2000"`
}

// ── Sentinel errors ───────────────────────────────────────────────────────────

var (
	ErrCommentNotFound = errors.New("comment not found")
	// ErrForbidden is already defined in post_service.go (same package)
)

// ── Interface ─────────────────────────────────────────────────────────────────

// CommentService handles comment CRUD and thread retrieval.
type CommentService interface {
	// CreateComment adds a top-level comment to a post.
	CreateComment(userID, postID uint, req *CreateCommentRequest) (*models.Comment, error)
	// CreateReply adds a reply to an existing comment.
	CreateReply(userID, postID, parentID uint, req *CreateCommentRequest) (*models.Comment, error)
	// ListComments returns top-level comments for a post (paginated).
	ListComments(postID uint, page, pageSize int) (*pagination.Page[models.Comment], error)
	// ListReplies returns threaded replies to a comment (paginated).
	ListReplies(commentID uint, page, pageSize int) (*pagination.Page[models.Comment], error)
	// UpdateComment edits a comment's content (owner only).
	UpdateComment(userID, commentID uint, req *UpdateCommentRequest) (*models.Comment, error)
	// DeleteComment soft-deletes a comment (owner only).
	DeleteComment(userID, commentID uint) error
}

// ── Implementation ────────────────────────────────────────────────────────────

type commentService struct {
	commentRepo repository.CommentRepository
	postRepo    repository.PostRepository
}

// NewCommentService constructs a CommentService.
func NewCommentService(commentRepo repository.CommentRepository, postRepo repository.PostRepository) CommentService {
	return &commentService{commentRepo: commentRepo, postRepo: postRepo}
}

func (s *commentService) CreateComment(userID, postID uint, req *CreateCommentRequest) (*models.Comment, error) {
	// Verify the post exists.
	if _, err := s.postRepo.FindByID(postID); errors.Is(err, repository.ErrNotFound) {
		return nil, ErrPostNotFound
	} else if err != nil {
		return nil, err
	}

	comment := &models.Comment{
		PostID:  postID,
		UserID:  userID,
		Content: req.Content,
	}
	if err := s.commentRepo.Create(comment); err != nil {
		return nil, err
	}

	// Best-effort: increment post's comment counter.
	_ = s.postRepo.IncrementCounter(postID, "comments_count", 1)

	return s.commentRepo.FindByIDWithAuthor(comment.ID)
}

func (s *commentService) CreateReply(userID, postID, parentID uint, req *CreateCommentRequest) (*models.Comment, error) {
	// Verify parent comment exists and belongs to the same post.
	parent, err := s.commentRepo.FindByID(parentID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrCommentNotFound
	}
	if err != nil {
		return nil, err
	}
	if parent.PostID != postID {
		return nil, ErrCommentNotFound // parent doesn't belong to this post
	}

	comment := &models.Comment{
		PostID:   postID,
		UserID:   userID,
		Content:  req.Content,
		ParentID: &parentID,
	}
	if err := s.commentRepo.Create(comment); err != nil {
		return nil, err
	}

	// Best-effort: bump the post's comment counter.
	_ = s.postRepo.IncrementCounter(postID, "comments_count", 1)

	return s.commentRepo.FindByIDWithAuthor(comment.ID)
}

func (s *commentService) ListComments(postID uint, page, pageSize int) (*pagination.Page[models.Comment], error) {
	offset := pagination.Offset(page, pageSize)
	comments, total, err := s.commentRepo.ListByPostID(postID, pageSize, offset)
	if err != nil {
		return nil, err
	}
	return pagination.New(comments, total, page, pageSize), nil
}

func (s *commentService) ListReplies(commentID uint, page, pageSize int) (*pagination.Page[models.Comment], error) {
	// Verify parent comment exists.
	if _, err := s.commentRepo.FindByID(commentID); errors.Is(err, repository.ErrNotFound) {
		return nil, ErrCommentNotFound
	} else if err != nil {
		return nil, err
	}

	offset := pagination.Offset(page, pageSize)
	replies, total, err := s.commentRepo.ListReplies(commentID, pageSize, offset)
	if err != nil {
		return nil, err
	}
	return pagination.New(replies, total, page, pageSize), nil
}

func (s *commentService) UpdateComment(userID, commentID uint, req *UpdateCommentRequest) (*models.Comment, error) {
	comment, err := s.commentRepo.FindByID(commentID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrCommentNotFound
	}
	if err != nil {
		return nil, err
	}
	if comment.UserID != userID {
		return nil, ErrForbidden
	}

	if err := s.commentRepo.UpdateContent(commentID, req.Content); err != nil {
		return nil, err
	}
	return s.commentRepo.FindByIDWithAuthor(commentID)
}

func (s *commentService) DeleteComment(userID, commentID uint) error {
	comment, err := s.commentRepo.FindByID(commentID)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrCommentNotFound
	}
	if err != nil {
		return err
	}
	if comment.UserID != userID {
		return ErrForbidden
	}

	if err := s.commentRepo.Delete(commentID); err != nil {
		return err
	}

	// Best-effort: decrement post's comment counter.
	_ = s.postRepo.IncrementCounter(comment.PostID, "comments_count", -1)
	return nil
}
