package services

import (
	"errors"
	"time"

	"github.com/Samueelx/g-nice-api/internal/models"
	"github.com/Samueelx/g-nice-api/internal/pagination"
	"github.com/Samueelx/g-nice-api/internal/repository"
)

// ── DTOs ─────────────────────────────────────────────────────────────────────

type SponsorRequest struct {
	Name       string  `json:"name" binding:"required,max=150"`
	LogoURL    *string `json:"logo_url" binding:"omitempty,url"`
	WebsiteURL *string `json:"website_url" binding:"omitempty,url"`
}

type CreateJokeRequest struct {
	Content    string          `json:"content" binding:"required,max=1000"`
	ActiveDate string          `json:"active_date" binding:"required"`
	Sponsor    *SponsorRequest `json:"sponsor"`
}

type UpdateJokeRequest struct {
	Content    *string         `json:"content" binding:"omitempty,max=1000"`
	ActiveDate *string         `json:"active_date" binding:"omitempty"`
	Sponsor    *SponsorRequest `json:"sponsor"`
	RemoveSponsor bool         `json:"-"` // We will set this in the handler if we see "sponsor": null
}

// ── Sentinel errors ───────────────────────────────────────────────────────────

var (
	ErrJokeNotFound = errors.New("joke not found")
	ErrDateTaken    = errors.New("a joke is already scheduled for this date")
	ErrInvalidDate  = errors.New("invalid date format, expected YYYY-MM-DD")
)

// ── Interface ─────────────────────────────────────────────────────────────────

type JokeService interface {
	GetTodayJoke() (*models.Joke, error)
	GetJoke(jokeID uint) (*models.Joke, error)
	ListJokes(page, pageSize int) (*pagination.Page[models.Joke], error)
	CreateJoke(userID uint, req *CreateJokeRequest) (*models.Joke, error)
	UpdateJoke(jokeID uint, req *UpdateJokeRequest) (*models.Joke, error)
	DeleteJoke(jokeID uint) error
	ToggleLike(jokeID, userID uint) (bool, int, error)
}

// ── Implementation ────────────────────────────────────────────────────────────

type jokeService struct {
	jokeRepo repository.JokeRepository
}

func NewJokeService(jokeRepo repository.JokeRepository) JokeService {
	return &jokeService{jokeRepo: jokeRepo}
}

func (s *jokeService) GetTodayJoke() (*models.Joke, error) {
	now := time.Now().UTC()
	joke, err := s.jokeRepo.GetByDate(now)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrJokeNotFound
	}
	return joke, err
}

func (s *jokeService) GetJoke(jokeID uint) (*models.Joke, error) {
	joke, err := s.jokeRepo.GetByID(jokeID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrJokeNotFound
	}
	return joke, err
}

func (s *jokeService) ListJokes(page, pageSize int) (*pagination.Page[models.Joke], error) {
	offset := pagination.Offset(page, pageSize)
	jokes, total, err := s.jokeRepo.ListAll(pageSize, offset)
	if err != nil {
		return nil, err
	}
	return pagination.New(jokes, total, page, pageSize), nil
}

func (s *jokeService) CreateJoke(userID uint, req *CreateJokeRequest) (*models.Joke, error) {
	activeDate, err := time.Parse("2006-01-02", req.ActiveDate)
	if err != nil {
		return nil, ErrInvalidDate
	}

	joke := &models.Joke{
		Content:     req.Content,
		ActiveDate:  activeDate,
		CreatedByID: userID,
	}

	if req.Sponsor != nil {
		joke.SponsorName = &req.Sponsor.Name
		joke.SponsorLogoURL = req.Sponsor.LogoURL
		joke.SponsorWebsiteURL = req.Sponsor.WebsiteURL
	}

	if err := s.jokeRepo.Create(joke); err != nil {
		if errors.Is(err, repository.ErrDuplicateKey) {
			return nil, ErrDateTaken
		}
		return nil, err
	}

	return s.jokeRepo.GetByID(joke.ID)
}

func (s *jokeService) UpdateJoke(jokeID uint, req *UpdateJokeRequest) (*models.Joke, error) {
	_, err := s.jokeRepo.GetByID(jokeID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrJokeNotFound
	}
	if err != nil {
		return nil, err
	}

	fields := make(map[string]interface{})
	if req.Content != nil {
		fields["content"] = *req.Content
	}
	if req.ActiveDate != nil {
		activeDate, err := time.Parse("2006-01-02", *req.ActiveDate)
		if err != nil {
			return nil, ErrInvalidDate
		}
		fields["active_date"] = activeDate
	}

	if req.RemoveSponsor {
		fields["sponsor_name"] = nil
		fields["sponsor_logo_url"] = nil
		fields["sponsor_website_url"] = nil
	} else if req.Sponsor != nil {
		fields["sponsor_name"] = req.Sponsor.Name
		if req.Sponsor.LogoURL != nil {
			fields["sponsor_logo_url"] = *req.Sponsor.LogoURL
		}
		if req.Sponsor.WebsiteURL != nil {
			fields["sponsor_website_url"] = *req.Sponsor.WebsiteURL
		}
	}

	if len(fields) > 0 {
		if err := s.jokeRepo.UpdateFields(jokeID, fields); err != nil {
			if errors.Is(err, repository.ErrDuplicateKey) {
				return nil, ErrDateTaken
			}
			return nil, err
		}
	}

	return s.jokeRepo.GetByID(jokeID)
}

func (s *jokeService) DeleteJoke(jokeID uint) error {
	err := s.jokeRepo.Delete(jokeID)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrJokeNotFound
	}
	return err
}

func (s *jokeService) ToggleLike(jokeID, userID uint) (bool, int, error) {
	_, err := s.jokeRepo.GetByID(jokeID)
	if errors.Is(err, repository.ErrNotFound) {
		return false, 0, ErrJokeNotFound
	}
	if err != nil {
		return false, 0, err
	}

	return s.jokeRepo.ToggleLike(jokeID, userID)
}
