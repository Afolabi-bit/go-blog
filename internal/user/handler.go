package user

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
	case errors.Is(err, ErrInvalidCredentials):
		response.Error(c, http.StatusUnauthorized, err.Error())
	case errors.Is(err, ErrEmailAlreadyExists), errors.Is(err, ErrInvalidInput):
		response.Error(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrUserNotFound):
		response.Error(c, http.StatusNotFound, err.Error())
	default:
		log.Printf("[ERROR] User handler unexpected error: %v", err)
		response.Error(c, http.StatusInternalServerError, "internal server error")
	}
}

func (h *Handler) Register(c *gin.Context) {
	var input RegisterRequest

	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid json payload")
		return
	}

	authResponse, err := h.svc.Register(c.Request.Context(), input)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusCreated, "User registered successfully", authResponse)
}

func (h *Handler) Login(c *gin.Context) {
	var input LoginRequest

	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid json payload")
		return
	}

	authResponse, err := h.svc.Login(c.Request.Context(), input)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "User logged in successfully", authResponse)
}

func (h *Handler) Me(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)

	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := h.svc.GetProfile(c.Request.Context(), userID)

	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "User profile fetched successfully", user)
}

func (h *Handler) Logout(c *gin.Context) {
	response.Success(c, http.StatusOK, "User logged out successfully", nil)
}
