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
	SearchAll   SearchType = "all"
	SearchUsers SearchType = "users"
	SearchPosts SearchType = "posts"
)

// SearchResult is the unified response envelope returned by SearchService.
// The Users and Posts fields are omitted from JSON when not requested.
type SearchResult struct {
	Query string                         `json:"query"`
	Type  SearchType                     `json:"type"`
	Users *pagination.Page[models.User]  `json:"users,omitempty"`
	Posts *pagination.Page[models.Post]  `json:"posts,omitempty"`
}

// ── Sentinel errors ───────────────────────────────────────────────────────────

var ErrSearchQueryEmpty = errors.New("search query must not be empty")

// ── Interface ─────────────────────────────────────────────────────────────────

// SearchService handles search business logic across users and posts.
type SearchService interface {
	// Search searches users and/or posts depending on searchType.
	// When searchType is SearchAll both repos are queried concurrently.
	Search(query string, searchType SearchType, page, pageSize int) (*SearchResult, error)
}

// ── Implementation ────────────────────────────────────────────────────────────

type searchService struct {
	userRepo repository.UserRepository
	postRepo repository.PostRepository
}

// NewSearchService constructs a SearchService.
func NewSearchService(userRepo repository.UserRepository, postRepo repository.PostRepository) SearchService {
	return &searchService{userRepo: userRepo, postRepo: postRepo}
}

func (s *searchService) Search(query string, searchType SearchType, page, pageSize int) (*SearchResult, error) {
	if query == "" {
		return nil, ErrSearchQueryEmpty
	}

	// Normalise searchType — default to "all" if an unknown value is given.
	switch searchType {
	case SearchAll, SearchUsers, SearchPosts:
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

	default: // SearchAll — fan out concurrently
		var (
			wg        sync.WaitGroup
			userPage  *pagination.Page[models.User]
			postPage  *pagination.Page[models.Post]
			userErr   error
			postErr   error
		)

		wg.Add(2)

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

		wg.Wait()

		if userErr != nil {
			return nil, userErr
		}
		if postErr != nil {
			return nil, postErr
		}

		result.Users = userPage
		result.Posts = postPage
	}

	return result, nil
}
