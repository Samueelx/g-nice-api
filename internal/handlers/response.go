package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// APIResponse is the standard envelope for all JSON responses.
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// OK sends a 200 response with data.
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: data})
}

// Created sends a 201 response with data.
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, APIResponse{Success: true, Data: data})
}

// BadRequest sends a 400 response with an error message.
func BadRequest(c *gin.Context, msg string) {
	c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: msg})
}

// Unauthorized sends a 401 response.
func Unauthorized(c *gin.Context, msg string) {
	c.JSON(http.StatusUnauthorized, APIResponse{Success: false, Error: msg})
}

// Forbidden sends a 403 response.
func Forbidden(c *gin.Context, msg string) {
	c.JSON(http.StatusForbidden, APIResponse{Success: false, Error: msg})
}

// NotFound sends a 404 response.
func NotFound(c *gin.Context, msg string) {
	c.JSON(http.StatusNotFound, APIResponse{Success: false, Error: msg})
}

// Conflict sends a 409 response (e.g. duplicate resource).
func Conflict(c *gin.Context, msg string) {
	c.JSON(http.StatusConflict, APIResponse{Success: false, Error: msg})
}

// InternalError sends a 500 response.
func InternalError(c *gin.Context) {
	c.JSON(http.StatusInternalServerError, APIResponse{
		Success: false,
		Error:   "an unexpected error occurred",
	})
}

// TooLarge sends a 413 response (file exceeds the size limit).
func TooLarge(c *gin.Context, msg string) {
	c.JSON(http.StatusRequestEntityTooLarge, APIResponse{Success: false, Error: msg})
}

// UnsupportedMediaType sends a 415 response (unacceptable file type).
func UnsupportedMediaType(c *gin.Context, msg string) {
	c.JSON(http.StatusUnsupportedMediaType, APIResponse{Success: false, Error: msg})
}
