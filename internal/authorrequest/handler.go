package authorrequest

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
	case errors.Is(err, ErrNotFound):
		response.Error(c, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrPendingExists):
		response.Error(c, http.StatusConflict, err.Error())
	case errors.Is(err, ErrAlreadyAuthor), errors.Is(err, ErrAlreadyProcessed):
		response.Error(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrInvalidInput), errors.Is(err, ErrInvalidID):
		response.Error(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrForbidden):
		response.Error(c, http.StatusForbidden, err.Error())
	default:
		log.Printf("[ERROR] AuthorRequest handler unexpected error: %v", err)
		response.Error(c, http.StatusInternalServerError, "internal server error")
	}
}

// Submit godoc
// @Summary      Submit application to become an author
// @Description  Allows an authenticated reader to apply for author status with a bio and sample portfolio links
// @Tags         author-requests
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body authorrequest.SubmitRequest true "Author application details"
// @Success      201  {object}  response.Response{data=authorrequest.AuthorRequest}
// @Failure      400  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Failure      409  {object}  response.Response
// @Router       /user/author-request [post]
func (h *Handler) Submit(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var input SubmitRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid json payload")
		return
	}

	req, err := h.svc.Submit(c.Request.Context(), userID, input)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusCreated, "Author request submitted successfully", req)
}

// GetMyRequest godoc
// @Summary      Get current user's author application
// @Description  Retrieves the authenticated user's latest author application and status
// @Tags         author-requests
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Response{data=authorrequest.AuthorRequest}
// @Failure      401  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Router       /user/author-request [get]
func (h *Handler) GetMyRequest(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	req, err := h.svc.GetMyRequest(c.Request.Context(), userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Author request retrieved successfully", req)
}

// List godoc
// @Summary      List all author applications
// @Description  Moderation list of author requests across all users (Admin only)
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Param        status query string false "Filter by status: pending, approved, rejected"
// @Param        limit  query int    false "Limit (default 10, max 20)"
// @Param        cursor query string false "Pagination cursor"
// @Success      200    {object}  response.Response
// @Failure      401    {object}  response.Response
// @Failure      403    {object}  response.Response
// @Router       /api/admin/author-requests [get]
func (h *Handler) List(c *gin.Context) {
	var filter Filter
	if err := c.ShouldBindQuery(&filter); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid query parameters")
		return
	}

	requests, meta, err := h.svc.ListRequests(c.Request.Context(), filter.Cursor, filter.Limit, filter)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Fetched author requests successfully", gin.H{
		"requests":   requests,
		"pagination": meta,
	})
}

// Review godoc
// @Summary      Review an author application
// @Description  Approve or reject a pending author request. Approving automatically elevates user to author role (Admin only).
// @Tags         admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id      path string                      true "Author Request ID"
// @Param        request body authorrequest.ReviewRequest true "Review decision and notes"
// @Success      200     {object} response.Response{data=authorrequest.AuthorRequest}
// @Failure      400     {object} response.Response
// @Failure      401     {object} response.Response
// @Failure      403     {object} response.Response
// @Failure      404     {object} response.Response
// @Router       /api/admin/author-requests/{id}/review [patch]
func (h *Handler) Review(c *gin.Context) {
	requestID := c.Param("id")

	reviewerID, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var input ReviewRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid json payload")
		return
	}

	updated, err := h.svc.ReviewRequest(c.Request.Context(), requestID, reviewerID, input)
	if err != nil {
		h.handleError(c, err)
		return
	}

	msg := "Author request reviewed successfully"
	if updated.Status == StatusApproved {
		msg = "Author request approved and user role elevated to author"
	} else if updated.Status == StatusRejected {
		msg = "Author request rejected"
	}

	response.Success(c, http.StatusOK, msg, updated)
}
