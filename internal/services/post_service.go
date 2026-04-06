package services

import (
	"errors"

	"github.com/Samueelx/g-nice-api/internal/models"
	"github.com/Samueelx/g-nice-api/internal/pagination"
	"github.com/Samueelx/g-nice-api/internal/repository"
)

// ── DTOs ─────────────────────────────────────────────────────────────────────

// CreatePostRequest is the payload for POST /api/v1/posts.
type CreatePostRequest struct {
	Content   string           `json:"content"    binding:"required,min=1,max=5000"`
	MediaURL  string           `json:"media_url"  binding:"omitempty,max=500"`
	MediaType models.MediaType `json:"media_type" binding:"omitempty,oneof=image video gif"`
	// IsPublic defaults to true if omitted.
	IsPublic *bool `json:"is_public"`
}

// UpdatePostRequest is the payload for PATCH /api/v1/posts/:id.
// Pointer fields: nil means "leave unchanged".
type UpdatePostRequest struct {
	Content  *string `json:"content"   binding:"omitempty,min=1,max=5000"`
	IsPublic *bool   `json:"is_public"`
}

// ── Sentinel errors ───────────────────────────────────────────────────────────

var (
	ErrPostNotFound = errors.New("post not found")
	ErrForbidden    = errors.New("you do not have permission to perform this action")
)

// ── Interface ─────────────────────────────────────────────────────────────────

// PostService handles all post-related business logic.
type PostService interface {
	CreatePost(userID uint, req *CreatePostRequest) (*models.Post, error)
	GetPost(postID uint) (*models.Post, error)
	UpdatePost(userID, postID uint, req *UpdatePostRequest) (*models.Post, error)
	DeletePost(userID, postID uint) error
	// ListFeed returns a paginated list of all public posts (chronological, newest first).
	ListFeed(page, pageSize int) (*pagination.Page[models.Post], error)
	// ListUserPosts returns a paginated list of public posts for the given username.
	ListUserPosts(username string, page, pageSize int) (*pagination.Page[models.Post], error)
}

// ── Implementation ────────────────────────────────────────────────────────────

type postService struct {
	postRepo repository.PostRepository
	userRepo repository.UserRepository
}

// NewPostService constructs a PostService.
func NewPostService(postRepo repository.PostRepository, userRepo repository.UserRepository) PostService {
	return &postService{postRepo: postRepo, userRepo: userRepo}
}

// CreatePost creates a new post and increments the author's post counter.
func (s *postService) CreatePost(userID uint, req *CreatePostRequest) (*models.Post, error) {
	isPublic := true
	if req.IsPublic != nil {
		isPublic = *req.IsPublic
	}

	post := &models.Post{
		UserID:    userID,
		Content:   req.Content,
		MediaURL:  req.MediaURL,
		MediaType: req.MediaType,
		IsPublic:  isPublic,
	}

	if err := s.postRepo.Create(post); err != nil {
		return nil, err
	}

	// Increment author's post count — best-effort (non-fatal if it fails)
	_ = s.userRepo.IncrementCounter(userID, "posts_count", 1)

	// Return with author preloaded so the frontend has everything it needs.
	return s.postRepo.FindByIDWithAuthor(post.ID)
}

// GetPost retrieves a single post with its author.
func (s *postService) GetPost(postID uint) (*models.Post, error) {
	post, err := s.postRepo.FindByIDWithAuthor(postID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrPostNotFound
	}
	return post, err
}

// UpdatePost applies a partial update, enforcing that only the owner can edit.
func (s *postService) UpdatePost(userID, postID uint, req *UpdatePostRequest) (*models.Post, error) {
	post, err := s.postRepo.FindByID(postID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrPostNotFound
	}
	if err != nil {
		return nil, err
	}
	if post.UserID != userID {
		return nil, ErrForbidden
	}

	fields := make(map[string]interface{})
	if req.Content != nil {
		fields["content"] = *req.Content
	}
	if req.IsPublic != nil {
		fields["is_public"] = *req.IsPublic
	}

	if len(fields) == 0 {
		return s.postRepo.FindByIDWithAuthor(postID)
	}

	if err := s.postRepo.UpdateFields(postID, fields); err != nil {
		return nil, err
	}

	return s.postRepo.FindByIDWithAuthor(postID)
}

// DeletePost soft-deletes a post, enforcing ownership, and decrements the counter.
func (s *postService) DeletePost(userID, postID uint) error {
	post, err := s.postRepo.FindByID(postID)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrPostNotFound
	}
	if err != nil {
		return err
	}
	if post.UserID != userID {
		return ErrForbidden
	}

	if err := s.postRepo.Delete(postID); err != nil {
		return err
	}

	// Decrement author's post count — best-effort
	_ = s.userRepo.IncrementCounter(userID, "posts_count", -1)
	return nil
}

// ListFeed returns a paginated, chronological feed of all public posts.
func (s *postService) ListFeed(page, pageSize int) (*pagination.Page[models.Post], error) {
	offset := pagination.Offset(page, pageSize)
	posts, total, err := s.postRepo.ListFeed(pageSize, offset)
	if err != nil {
		return nil, err
	}
	return pagination.New(posts, total, page, pageSize), nil
}

// ListUserPosts returns a paginated list of public posts for the given username.
func (s *postService) ListUserPosts(username string, page, pageSize int) (*pagination.Page[models.Post], error) {
	user, err := s.userRepo.FindByUsername(username)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrProfileNotFound
	}
	if err != nil {
		return nil, err
	}

	offset := pagination.Offset(page, pageSize)
	posts, total, err := s.postRepo.ListByUserID(user.ID, pageSize, offset)
	if err != nil {
		return nil, err
	}
	return pagination.New(posts, total, page, pageSize), nil
}
