package likes

import (
	"blog-api/internal/middleware"
	"blog-api/internal/response"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

func NewHandler(s *Service) *Handler {
	return &Handler{svc: s}
}

func (h *Handler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrPostNotFound):
		response.Error(c, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrInvalidID):
		response.Error(c, http.StatusBadRequest, err.Error())
	default:
		log.Printf("[ERROR] Likes handler unexpected error: %v", err)
		response.Error(c, http.StatusInternalServerError, "internal server error")
	}
}

// ToggleLike godoc
// @Summary      Toggle like on a post
// @Description  Likes a post if not currently liked, or removes the like if already liked (authenticated users)
// @Tags         posts
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Post ID"
// @Success      200 {object} response.Response{data=likes.LikeStatus}
// @Failure      400 {object} response.Response
// @Failure      401 {object} response.Response
// @Failure      404 {object} response.Response
// @Router       /api/posts/{id}/like [post]
func (h *Handler) ToggleLike(c *gin.Context) {
	postID := c.Param("id")

	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	status, err := h.svc.ToggleLike(c.Request.Context(), postID, userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	msg := "Post liked"
	if !status.Liked {
		msg = "Post unliked"
	}

	response.Success(c, http.StatusOK, msg, status)
}

// GetStatus godoc
// @Summary      Get like status for a post
// @Description  Check whether the current user liked the post and get total likes count
// @Tags         posts
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Post ID"
// @Success      200 {object} response.Response{data=likes.LikeStatus}
// @Failure      400 {object} response.Response
// @Failure      404 {object} response.Response
// @Router       /api/posts/{id}/like [get]
func (h *Handler) GetStatus(c *gin.Context) {
	postID := c.Param("id")

	var userIDPtr *string
	if userID, ok := middleware.GetUserID(c); ok {
		userIDPtr = &userID
	}

	status, err := h.svc.GetStatus(c.Request.Context(), postID, userIDPtr)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Like status fetched successfully", status)
}
