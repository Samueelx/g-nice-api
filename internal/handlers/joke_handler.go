package handlers

import (
	"errors"
	"log"
	"strconv"

	"github.com/Samueelx/g-nice-api/internal/models"
	"github.com/Samueelx/g-nice-api/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

type JokeHandler struct {
	jokeSvc services.JokeService
}

func NewJokeHandler(jokeSvc services.JokeService) *JokeHandler {
	return &JokeHandler{jokeSvc: jokeSvc}
}

// GetTodayJoke GET /api/v1/jokes/today
func (h *JokeHandler) GetTodayJoke(c *gin.Context) {
	joke, err := h.jokeSvc.GetTodayJoke()
	if err != nil {
		if errors.Is(err, services.ErrJokeNotFound) {
			OKWithMsg(c, nil, "No joke scheduled for today")
			return
		}
		log.Printf("GetTodayJoke error: %v", err)
		InternalError(c)
		return
	}

	userID, ok := extractUserIDOptional(c)
	if ok && userID > 0 {
		// Populate is_liked logic if needed or let the frontend compute based on a separate endpoint/field.
		// Since the prompt asks for is_liked populated for auth users, we would need to ask the service.
		// For simplicity, if we don't have a direct query in the service, we could check if the user is in the likes array.
		joke.IsLiked = checkUserLiked(joke.Likes, userID)
	}

	OK(c, joke)
}

// GetJoke GET /api/v1/jokes/:id
func (h *JokeHandler) GetJoke(c *gin.Context) {
	jokeID, ok := parseJokeID(c, "id")
	if !ok {
		return
	}

	joke, err := h.jokeSvc.GetJoke(jokeID)
	if err != nil {
		if errors.Is(err, services.ErrJokeNotFound) {
			NotFound(c, "joke not found")
			return
		}
		log.Printf("GetJoke error: %v", err)
		InternalError(c)
		return
	}

	userID, ok := extractUserIDOptional(c)
	if ok && userID > 0 {
		joke.IsLiked = checkUserLiked(joke.Likes, userID)
	}

	OK(c, joke)
}

// ListJokes GET /api/v1/admin/jokes
func (h *JokeHandler) ListJokes(c *gin.Context) {
	page, pageSize := parsePagination(c)
	
	result, err := h.jokeSvc.ListJokes(page, pageSize)
	if err != nil {
		log.Printf("ListJokes error: %v", err)
		InternalError(c)
		return
	}

	OK(c, result)
}

// CreateJoke POST /api/v1/admin/jokes
func (h *JokeHandler) CreateJoke(c *gin.Context) {
	userID, ok := extractUserID(c)
	if !ok {
		return
	}

	var req services.CreateJokeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	joke, err := h.jokeSvc.CreateJoke(userID, &req)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrDateTaken):
			Conflict(c, err.Error())
		case errors.Is(err, services.ErrInvalidDate):
			BadRequest(c, err.Error())
		default:
			log.Printf("CreateJoke error: %v", err)
			InternalError(c)
		}
		return
	}

	Created(c, joke)
}

// UpdateJoke PATCH /api/v1/admin/jokes/:id
func (h *JokeHandler) UpdateJoke(c *gin.Context) {
	jokeID, ok := parseJokeID(c, "id")
	if !ok {
		return
	}

	var req services.UpdateJokeRequest
	if err := c.ShouldBindBodyWith(&req, binding.JSON); err != nil {
		BadRequest(c, err.Error())
		return
	}

	var rawMap map[string]interface{}
	if err := c.ShouldBindBodyWith(&rawMap, binding.JSON); err == nil {
		if sponsorVal, exists := rawMap["sponsor"]; exists && sponsorVal == nil {
			req.RemoveSponsor = true
		}
	}

	joke, err := h.jokeSvc.UpdateJoke(jokeID, &req)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrJokeNotFound):
			NotFound(c, "joke not found")
		case errors.Is(err, services.ErrDateTaken):
			Conflict(c, err.Error())
		case errors.Is(err, services.ErrInvalidDate):
			BadRequest(c, err.Error())
		default:
			log.Printf("UpdateJoke error: %v", err)
			InternalError(c)
		}
		return
	}

	OK(c, joke)
}

// DeleteJoke DELETE /api/v1/admin/jokes/:id
func (h *JokeHandler) DeleteJoke(c *gin.Context) {
	jokeID, ok := parseJokeID(c, "id")
	if !ok {
		return
	}

	if err := h.jokeSvc.DeleteJoke(jokeID); err != nil {
		if errors.Is(err, services.ErrJokeNotFound) {
			NotFound(c, "joke not found")
			return
		}
		log.Printf("DeleteJoke error: %v", err)
		InternalError(c)
		return
	}

	ok204(c)
}

// ToggleLike POST /api/v1/jokes/:id/like
func (h *JokeHandler) ToggleLike(c *gin.Context) {
	userID, ok := extractUserID(c)
	if !ok {
		return
	}
	jokeID, ok := parseJokeID(c, "id")
	if !ok {
		return
	}

	liked, count, err := h.jokeSvc.ToggleLike(jokeID, userID)
	if err != nil {
		if errors.Is(err, services.ErrJokeNotFound) {
			NotFound(c, "joke not found")
			return
		}
		log.Printf("ToggleLike error: %v", err)
		InternalError(c)
		return
	}

	OK(c, gin.H{
		"liked":       liked,
		"likes_count": count,
	})
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func parseJokeID(c *gin.Context, param string) (uint, bool) {
	id, err := strconv.ParseUint(c.Param(param), 10, 64)
	if err != nil || id == 0 {
		BadRequest(c, "invalid ID")
		return 0, false
	}
	return uint(id), true
}

func OKWithMsg(c *gin.Context, data interface{}, msg string) {
	c.JSON(200, gin.H{
		"success": true,
		"data":    data,
		"message": msg,
	})
}

// extractUserIDOptional tries to get the user ID if authorized, without failing
func extractUserIDOptional(c *gin.Context) (uint, bool) {
	val, exists := c.Get("userID")
	if !exists {
		return 0, false
	}
	id, ok := val.(uint)
	return id, ok
}

func checkUserLiked(likes []models.JokeLike, userID uint) bool {
	for _, l := range likes {
		if l.UserID == userID {
			return true
		}
	}
	return false
}
