package media

import (
	"blog-api/internal/response"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

func NewHandler(s *Service) *Handler {
	return &Handler{svc: s}
}

// Upload godoc
// @Summary      Upload an image asset
// @Description  Upload an image file (cover image, inline asset) up to 5MB (Author or Admin role required)
// @Tags         media
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        file formData file true "Image file to upload (JPEG, PNG, WebP, GIF)"
// @Success      201  {object} response.Response{data=media.UploadResponse}
// @Failure      400  {object} response.Response
// @Failure      401  {object} response.Response
// @Failure      403  {object} response.Response
// @Router       /api/media/upload [post]
func (h *Handler) Upload(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		// Also check "cover" or "image"
		file, header, err = c.Request.FormFile("cover")
		if err != nil {
			file, header, err = c.Request.FormFile("image")
			if err != nil {
				response.Error(c, http.StatusBadRequest, "no image file provided (expected 'file', 'cover', or 'image')")
				return
			}
		}
	}
	defer file.Close()

	uploadRes, err := h.svc.SaveFile(c.Request.Context(), file, header)
	if err != nil {
		switch {
		case errors.Is(err, ErrFileTooLarge), errors.Is(err, ErrInvalidFileType):
			response.Error(c, http.StatusBadRequest, err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, "failed to save uploaded file")
		}
		return
	}

	response.Success(c, http.StatusCreated, "Image uploaded successfully", uploadRes)
}
