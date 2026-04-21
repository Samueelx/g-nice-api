package handlers

import (
	"errors"
	"log"

	"github.com/Samueelx/g-nice-api/internal/services"
	"github.com/gin-gonic/gin"
)

// allowedPrefixes restricts the ?type query param to known S3 key prefixes.
var allowedPrefixes = map[string]bool{
	"posts":   true,
	"avatars": true,
}

// UploadHandler handles media file upload requests.
type UploadHandler struct {
	uploadSvc services.UploadService
}

// NewUploadHandler constructs an UploadHandler.
func NewUploadHandler(uploadSvc services.UploadService) *UploadHandler {
	return &UploadHandler{uploadSvc: uploadSvc}
}

// Upload godoc
// POST /api/v1/uploads?type=posts|avatars
// Content-Type: multipart/form-data
// Form field: "file"
//
// Returns: { "success": true, "data": { "url": "https://..." } }
func (h *UploadHandler) Upload(c *gin.Context) {
	userID, ok := extractUserID(c)
	if !ok {
		return
	}

	// Determine the S3 key prefix from the optional ?type query param.
	prefix := c.DefaultQuery("type", "posts")
	if !allowedPrefixes[prefix] {
		BadRequest(c, "invalid type; allowed values: posts, avatars")
		return
	}

	// Pull the uploaded file from the multipart form.
	fileHeader, err := c.FormFile("file")
	if err != nil {
		BadRequest(c, "file field is required")
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		log.Printf("Upload: failed to open multipart file: %v", err)
		InternalError(c)
		return
	}
	defer file.Close()

	url, err := h.uploadSvc.UploadMedia(c.Request.Context(), userID, prefix, file, fileHeader)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrFileTooLarge):
			TooLarge(c, err.Error())
		case errors.Is(err, services.ErrUnsupportedMediaType):
			UnsupportedMediaType(c, err.Error())
		default:
			log.Printf("Upload error: %v", err)
			InternalError(c)
		}
		return
	}

	OK(c, gin.H{"url": url})
}
