package comments

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
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrPostNotFound), errors.Is(err, ErrParentNotFound):
		response.Error(c, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrForbidden):
		response.Error(c, http.StatusForbidden, err.Error())
	case errors.Is(err, ErrInvalidInput), errors.Is(err, ErrInvalidID):
		response.Error(c, http.StatusBadRequest, err.Error())
	default:
		log.Printf("[ERROR] Comments handler unexpected error: %v", err)
		response.Error(c, http.StatusInternalServerError, "internal server error")
	}
}

// AddComment godoc
// @Summary      Add a comment to a post
// @Description  Add a new comment or nested reply to a published post (authenticated users)
// @Tags         comments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id      path string                        true "Post ID"
// @Param        request body comments.CreateCommentRequest true "Comment payload"
// @Success      201     {object} response.Response{data=comments.Comment}
// @Failure      400     {object} response.Response
// @Failure      401     {object} response.Response
// @Failure      404     {object} response.Response
// @Router       /api/posts/{id}/comments [post]
func (h *Handler) AddComment(c *gin.Context) {
	postID := c.Param("id")

	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var input CreateCommentRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid json payload")
		return
	}

	comment, err := h.svc.AddComment(c.Request.Context(), postID, userID, input)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusCreated, "Comment added successfully", comment)
}

// ListComments godoc
// @Summary      List comments on a post
// @Description  Browse paginated comments on a published post
// @Tags         comments
// @Produce      json
// @Param        id     path  string false "Post ID"
// @Param        limit  query int    false "Number of comments to return (default 10, max 20)"
// @Param        cursor query string false "Pagination cursor"
// @Success      200    {object} response.Response
// @Failure      400    {object} response.Response
// @Failure      404    {object} response.Response
// @Router       /api/posts/{id}/comments [get]
func (h *Handler) ListComments(c *gin.Context) {
	postID := c.Param("id")

	var filter CommentFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid query parameters")
		return
	}

	commentsList, meta, err := h.svc.ListComments(c.Request.Context(), postID, filter.Cursor, filter.Limit)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Comments fetched successfully", gin.H{
		"comments":   commentsList,
		"pagination": meta,
	})
}

// DeleteComment godoc
// @Summary      Delete a comment
// @Description  Deletes a comment (Comment author, Post author, or Admin)
// @Tags         comments
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Comment ID"
// @Success      200 {object} response.Response
// @Failure      400 {object} response.Response
// @Failure      401 {object} response.Response
// @Failure      403 {object} response.Response
// @Failure      404 {object} response.Response
// @Router       /api/comments/{id} [delete]
func (h *Handler) DeleteComment(c *gin.Context) {
	commentID := c.Param("id")

	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	role, ok := middleware.GetRole(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := h.svc.DeleteComment(c.Request.Context(), commentID, userID, role); err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Comment deleted successfully", nil)
}
