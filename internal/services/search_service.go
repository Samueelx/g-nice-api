package services

import (
	"errors"
	"sync"

	"github.com/Samueelx/g-nice-api/internal/models"
	"github.com/Samueelx/g-nice-api/internal/pagination"
	"github.com/Samueelx/g-nice-api/internal/repository"
)

// ── Types ─────────────────────────────────────────────────────────────────────

// SearchType controls which resource types are searched.
type SearchType string

const (
	SearchAll    SearchType = "all"
	SearchUsers  SearchType = "users"
	SearchPosts  SearchType = "posts"
	SearchEvents SearchType = "events"
)

// SearchResult is the unified response envelope returned by SearchService.
// The Users, Posts, and Events fields are omitted from JSON when not requested.
type SearchResult struct {
	Query  string                          `json:"query"`
	Type   SearchType                      `json:"type"`
	Users  *pagination.Page[models.User]   `json:"users,omitempty"`
	Posts  *pagination.Page[models.Post]   `json:"posts,omitempty"`
	Events *pagination.Page[models.Event]  `json:"events,omitempty"`
}

// ── Sentinel errors ───────────────────────────────────────────────────────────

var ErrSearchQueryEmpty = errors.New("search query must not be empty")

// ── Interface ─────────────────────────────────────────────────────────────────

// SearchService handles search business logic across users, posts, and events.
type SearchService interface {
	// Search searches users, posts, and/or events depending on searchType.
	// When searchType is SearchAll all three repos are queried concurrently.
	Search(query string, searchType SearchType, page, pageSize int) (*SearchResult, error)
}

// ── Implementation ────────────────────────────────────────────────────────────

type searchService struct {
	userRepo  repository.UserRepository
	postRepo  repository.PostRepository
	eventRepo repository.EventRepository
}

// NewSearchService constructs a SearchService.
func NewSearchService(userRepo repository.UserRepository, postRepo repository.PostRepository, eventRepo repository.EventRepository) SearchService {
	return &searchService{userRepo: userRepo, postRepo: postRepo, eventRepo: eventRepo}
}

func (s *searchService) Search(query string, searchType SearchType, page, pageSize int) (*SearchResult, error) {
	if query == "" {
		return nil, ErrSearchQueryEmpty
	}

	// Normalise searchType — default to "all" if an unknown value is given.
	switch searchType {
	case SearchAll, SearchUsers, SearchPosts, SearchEvents:
		// valid
	default:
		searchType = SearchAll
	}

	offset := pagination.Offset(page, pageSize)

	result := &SearchResult{
		Query: query,
		Type:  searchType,
	}

	switch searchType {
	case SearchUsers:
		users, total, err := s.userRepo.Search(query, pageSize, offset)
		if err != nil {
			return nil, err
		}
		result.Users = pagination.New(users, total, page, pageSize)

	case SearchPosts:
		posts, total, err := s.postRepo.Search(query, pageSize, offset)
		if err != nil {
			return nil, err
		}
		result.Posts = pagination.New(posts, total, page, pageSize)

	case SearchEvents:
		events, total, err := s.eventRepo.List(
			repository.EventFilter{Search: query}, pageSize, offset,
		)
		if err != nil {
			return nil, err
		}
		result.Events = pagination.New(events, total, page, pageSize)

	default: // SearchAll — fan out concurrently
		var (
			wg         sync.WaitGroup
			userPage   *pagination.Page[models.User]
			postPage   *pagination.Page[models.Post]
			eventPage  *pagination.Page[models.Event]
			userErr    error
			postErr    error
			eventErr   error
		)

		wg.Add(3)

		go func() {
			defer wg.Done()
			users, total, err := s.userRepo.Search(query, pageSize, offset)
			if err != nil {
				userErr = err
				return
			}
			userPage = pagination.New(users, total, page, pageSize)
		}()

		go func() {
			defer wg.Done()
			posts, total, err := s.postRepo.Search(query, pageSize, offset)
			if err != nil {
				postErr = err
				return
			}
			postPage = pagination.New(posts, total, page, pageSize)
		}()

		go func() {
			defer wg.Done()
			events, total, err := s.eventRepo.List(
				repository.EventFilter{Search: query}, pageSize, offset,
			)
			if err != nil {
				eventErr = err
				return
			}
			eventPage = pagination.New(events, total, page, pageSize)
		}()

		wg.Wait()

		if userErr != nil {
			return nil, userErr
		}
		if postErr != nil {
			return nil, postErr
		}
		if eventErr != nil {
			return nil, eventErr
		}

		result.Users  = userPage
		result.Posts  = postPage
		result.Events = eventPage
	}

	return result, nil
}
