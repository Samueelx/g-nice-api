package services

import (
	"errors"

	"github.com/Samueelx/g-nice-api/internal/models"
	"github.com/Samueelx/g-nice-api/internal/pagination"
	"github.com/Samueelx/g-nice-api/internal/repository"
)

// ── DTOs ─────────────────────────────────────────────────────────────────────

type CreateJokeCommentRequest struct {
	Content string `json:"content" binding:"required,max=500"`
}

type UpdateJokeCommentRequest struct {
	Content *string `json:"content" binding:"required,max=500"`
}

// ── Sentinel errors ───────────────────────────────────────────────────────────

var (
	ErrJokeCommentNotFound = errors.New("joke comment not found")
)

// ── Interface ─────────────────────────────────────────────────────────────────

type JokeCommentService interface {
	ListComments(jokeID uint, page, pageSize int) (*pagination.Page[models.JokeComment], error)
	ListReplies(parentID uint, page, pageSize int) (*pagination.Page[models.JokeComment], error)
	CreateComment(jokeID, userID uint, req *CreateJokeCommentRequest) (*models.JokeComment, error)
	CreateReply(jokeID, parentID, userID uint, req *CreateJokeCommentRequest) (*models.JokeComment, error)
	UpdateComment(commentID, userID uint, req *UpdateJokeCommentRequest) (*models.JokeComment, error)
	DeleteComment(commentID, userID uint, isAdmin bool) error
	ToggleLike(commentID, userID uint) (bool, int, error)
}

// ── Implementation ────────────────────────────────────────────────────────────

type jokeCommentService struct {
	commentRepo repository.JokeCommentRepository
	jokeRepo    repository.JokeRepository
	notifSvc    NotificationService
	userRepo    repository.UserRepository
}

func NewJokeCommentService(
	commentRepo repository.JokeCommentRepository,
	jokeRepo repository.JokeRepository,
	notifSvc NotificationService,
	userRepo repository.UserRepository,
) JokeCommentService {
	return &jokeCommentService{commentRepo: commentRepo, jokeRepo: jokeRepo, notifSvc: notifSvc, userRepo: userRepo}
}

func (s *jokeCommentService) ListComments(jokeID uint, page, pageSize int) (*pagination.Page[models.JokeComment], error) {
	_, err := s.jokeRepo.GetByID(jokeID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrJokeNotFound
	}

	offset := pagination.Offset(page, pageSize)
	comments, total, err := s.commentRepo.ListComments(jokeID, pageSize, offset)
	if err != nil {
		return nil, err
	}
	return pagination.New(comments, total, page, pageSize), nil
}

func (s *jokeCommentService) ListReplies(parentID uint, page, pageSize int) (*pagination.Page[models.JokeComment], error) {
	_, err := s.commentRepo.GetByID(parentID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrJokeCommentNotFound
	}

	offset := pagination.Offset(page, pageSize)
	comments, total, err := s.commentRepo.ListReplies(parentID, pageSize, offset)
	if err != nil {
		return nil, err
	}
	return pagination.New(comments, total, page, pageSize), nil
}

func (s *jokeCommentService) CreateComment(jokeID, userID uint, req *CreateJokeCommentRequest) (*models.JokeComment, error) {
	_, err := s.jokeRepo.GetByID(jokeID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrJokeNotFound
	}
	if err != nil {
		return nil, err
	}

	comment := &models.JokeComment{
		JokeID:  jokeID,
		UserID:  userID,
		Content: req.Content,
	}

	if err := s.commentRepo.Create(comment); err != nil {
		return nil, err
	}

	_ = s.jokeRepo.IncrementCommentCount(jokeID, 1)

	// Dispatch mention notifications off the request path — best-effort.
	commentID := comment.ID
	go dispatchMentions(req.Content, userID, &commentID, "joke_comment", s.userRepo, s.notifSvc)

	return s.commentRepo.GetByID(comment.ID)
}

func (s *jokeCommentService) CreateReply(jokeID, parentID, userID uint, req *CreateJokeCommentRequest) (*models.JokeComment, error) {
	_, err := s.jokeRepo.GetByID(jokeID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrJokeNotFound
	}

	parent, err := s.commentRepo.GetByID(parentID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrJokeCommentNotFound
	}
	if err != nil {
		return nil, err
	}
	if parent.JokeID != jokeID {
		return nil, errors.New("parent comment does not belong to this joke")
	}

	comment := &models.JokeComment{
		JokeID:   jokeID,
		UserID:   userID,
		ParentID: &parentID,
		Content:  req.Content,
	}

	if err := s.commentRepo.Create(comment); err != nil {
		return nil, err
	}

	_ = s.commentRepo.IncrementReplyCount(parentID, 1)

	// Dispatch mention notifications off the request path — best-effort.
	commentID := comment.ID
	go dispatchMentions(req.Content, userID, &commentID, "joke_comment", s.userRepo, s.notifSvc)

	return s.commentRepo.GetByID(comment.ID)
}

func (s *jokeCommentService) UpdateComment(commentID, userID uint, req *UpdateJokeCommentRequest) (*models.JokeComment, error) {
	comment, err := s.commentRepo.GetByID(commentID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrJokeCommentNotFound
	}
	if err != nil {
		return nil, err
	}
	if comment.UserID != userID {
		return nil, ErrForbidden
	}

	fields := make(map[string]interface{})
	if req.Content != nil {
		fields["content"] = *req.Content
	}

	if len(fields) > 0 {
		if err := s.commentRepo.UpdateFields(commentID, fields); err != nil {
			return nil, err
		}
	}

	return s.commentRepo.GetByID(commentID)
}

func (s *jokeCommentService) DeleteComment(commentID, userID uint, isAdmin bool) error {
	comment, err := s.commentRepo.GetByID(commentID)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrJokeCommentNotFound
	}
	if err != nil {
		return err
	}
	
	if comment.UserID != userID && !isAdmin {
		return ErrForbidden
	}

	if err := s.commentRepo.Delete(commentID); err != nil {
		return err
	}

	if comment.ParentID == nil {
		_ = s.jokeRepo.IncrementCommentCount(comment.JokeID, -1)
	} else {
		_ = s.commentRepo.IncrementReplyCount(*comment.ParentID, -1)
	}

	return nil
}

func (s *jokeCommentService) ToggleLike(commentID, userID uint) (bool, int, error) {
	_, err := s.commentRepo.GetByID(commentID)
	if errors.Is(err, repository.ErrNotFound) {
		return false, 0, ErrJokeCommentNotFound
	}
	if err != nil {
		return false, 0, err
	}

	return s.commentRepo.ToggleLike(commentID, userID)
}
