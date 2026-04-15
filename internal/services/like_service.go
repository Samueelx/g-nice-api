package services

import (
	"errors"

	"github.com/Samueelx/g-nice-api/internal/models"
	"github.com/Samueelx/g-nice-api/internal/repository"
)

// LikeResult describes the outcome of a toggle operation.
type LikeResult struct {
	Liked      bool `json:"liked"`       // true = liked, false = unliked
	LikesCount int  `json:"likes_count"` // updated count on the target entity
}

// ── Sentinel errors ───────────────────────────────────────────────────────────

var ErrLikeTargetNotFound = errors.New("target not found")

// ── Interface ─────────────────────────────────────────────────────────────────

// LikeService handles toggling likes on posts and comments.
type LikeService interface {
	// TogglePostLike likes or unlikes a post and returns the new state.
	TogglePostLike(userID, postID uint) (*LikeResult, error)
	// ToggleCommentLike likes or unlikes a comment and returns the new state.
	ToggleCommentLike(userID, commentID uint) (*LikeResult, error)
}

// ── Implementation ────────────────────────────────────────────────────────────

type likeService struct {
	likeRepo    repository.LikeRepository
	postRepo    repository.PostRepository
	commentRepo repository.CommentRepository
	notifSvc    NotificationService
}

// NewLikeService constructs a LikeService.
func NewLikeService(
	likeRepo repository.LikeRepository,
	postRepo repository.PostRepository,
	commentRepo repository.CommentRepository,
	notifSvc NotificationService,
) LikeService {
	return &likeService{
		likeRepo:    likeRepo,
		postRepo:    postRepo,
		commentRepo: commentRepo,
		notifSvc:    notifSvc,
	}
}

// TogglePostLike adds a like if none exists, or removes it if it does.
// Returns the updated liked state and current likes_count.
func (s *likeService) TogglePostLike(userID, postID uint) (*LikeResult, error) {
	// Verify the post exists and fetch current count.
	post, err := s.postRepo.FindByID(postID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrLikeTargetNotFound
	}
	if err != nil {
		return nil, err
	}

	existing, err := s.likeRepo.FindByUserAndTarget(userID, postID, models.LikeTargetPost)

	if errors.Is(err, repository.ErrNotFound) {
		// Not yet liked → create like.
		like := &models.Like{
			UserID:     userID,
			TargetID:   postID,
			TargetType: models.LikeTargetPost,
		}
		if err := s.likeRepo.Create(like); err != nil {
			return nil, err
		}
		_ = s.postRepo.IncrementCounter(postID, "likes_count", 1)
		// Best-effort: notify the post author.
		_ = s.notifSvc.Notify(post.UserID, userID, models.NotifTypeLikePost, &postID, "post")
		return &LikeResult{Liked: true, LikesCount: post.LikesCount + 1}, nil
	}
	if err != nil {
		return nil, err
	}

	// Already liked → remove like.
	if err := s.likeRepo.Delete(existing.ID); err != nil {
		return nil, err
	}
	_ = s.postRepo.IncrementCounter(postID, "likes_count", -1)

	count := post.LikesCount - 1
	if count < 0 {
		count = 0
	}
	return &LikeResult{Liked: false, LikesCount: count}, nil
}

// ToggleCommentLike adds a like if none exists, or removes it if it does.
func (s *likeService) ToggleCommentLike(userID, commentID uint) (*LikeResult, error) {
	// Verify the comment exists.
	comment, err := s.commentRepo.FindByID(commentID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrLikeTargetNotFound
	}
	if err != nil {
		return nil, err
	}

	existing, err := s.likeRepo.FindByUserAndTarget(userID, commentID, models.LikeTargetComment)

	if errors.Is(err, repository.ErrNotFound) {
		like := &models.Like{
			UserID:     userID,
			TargetID:   commentID,
			TargetType: models.LikeTargetComment,
		}
		if err := s.likeRepo.Create(like); err != nil {
			return nil, err
		}
		_ = s.commentRepo.IncrementCounter(commentID, "likes_count", 1)
		// Best-effort: notify the comment author.
		_ = s.notifSvc.Notify(comment.UserID, userID, models.NotifTypeLikeComment, &commentID, "comment")
		return &LikeResult{Liked: true, LikesCount: comment.LikesCount + 1}, nil
	}
	if err != nil {
		return nil, err
	}

	if err := s.likeRepo.Delete(existing.ID); err != nil {
		return nil, err
	}
	_ = s.commentRepo.IncrementCounter(commentID, "likes_count", -1)

	count := comment.LikesCount - 1
	if count < 0 {
		count = 0
	}
	return &LikeResult{Liked: false, LikesCount: count}, nil
}
