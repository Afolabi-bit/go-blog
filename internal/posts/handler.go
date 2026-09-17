package posts

import (
	"blog-api/internal/middleware"
	"blog-api/internal/response"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

func NewHandler(s *Service) *Handler {
	return &Handler{
		svc: s,
	}
}

func (h *Handler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		response.Error(c, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrForbidden):
		response.Error(c, http.StatusForbidden, err.Error())
	case errors.Is(err, ErrInvalidInput):
		response.Error(c, http.StatusBadRequest, err.Error())
	default:
		response.Error(c, http.StatusInternalServerError, "internal server error")
	}
}

func (h *Handler) CreatePost(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)

	if !ok {
		response.Error(c, http.StatusUnauthorized, "Please login to create post.")
		return
	}

	user, err := h.svc.user.FinduserByID(c, userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	var post CreatePostRequest

	if err = c.ShouldBindJSON(&post); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	authorName := user.FirstName + " " + user.LastName
	createdPost, err := h.svc.CreateNewPost(c, user.ID, authorName, post)

	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusCreated, "post successfully created", createdPost)
}
