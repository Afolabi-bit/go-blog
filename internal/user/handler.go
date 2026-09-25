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

// Register godoc
// @Summary      Register a new user
// @Description  Creates a new user account (defaults to reader role)
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body user.RegisterRequest true "User registration details"
// @Success      201  {object}  response.Response{data=user.AuthResponse}
// @Failure      400  {object}  response.Response
// @Router       /auth/register [post]
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

// Login godoc
// @Summary      Log in user
// @Description  Authenticates user credentials and returns a JWT Bearer token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body user.LoginRequest true "Login credentials"
// @Success      200  {object}  response.Response{data=user.AuthResponse}
// @Failure      400  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Router       /auth/login [post]
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

// Me godoc
// @Summary      Get current user profile
// @Description  Returns details of the currently authenticated user
// @Tags         user
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Response{data=user.PublicUser}
// @Failure      401  {object}  response.Response
// @Router       /user/iam [get]
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

// Logout godoc
// @Summary      Log out user
// @Description  Logs out the user session and revokes refresh tokens
// @Tags         auth
// @Produce      json
// @Security     BearerAuth
// @Param        request body user.RefreshTokenRequest false "Optional refresh token to revoke"
// @Success      200  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Router       /auth/logout [post]
func (h *Handler) Logout(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var input RefreshTokenRequest
	var tokenPtr *string
	if err := c.ShouldBindJSON(&input); err == nil && input.RefreshToken != "" {
		tokenPtr = &input.RefreshToken
	}

	_ = h.svc.Logout(c.Request.Context(), userID, tokenPtr)
	response.Success(c, http.StatusOK, "User logged out successfully", nil)
}

// RefreshToken godoc
// @Summary      Refresh access token
// @Description  Exchanges an active refresh token for a new access token and rotated refresh token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body user.RefreshTokenRequest true "Refresh token payload"
// @Success      200  {object}  response.Response{data=user.AuthResponse}
// @Failure      400  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Router       /auth/refresh [post]
func (h *Handler) RefreshToken(c *gin.Context) {
	var input RefreshTokenRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid json payload")
		return
	}

	authResponse, err := h.svc.RefreshToken(c.Request.Context(), input.RefreshToken)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "invalid or expired refresh token")
		return
	}

	response.Success(c, http.StatusOK, "Token refreshed successfully", authResponse)
}

// UpdateProfile godoc
// @Summary      Update user profile
// @Description  Update current user's profile details (first name, last name, bio, avatar)
// @Tags         user
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body user.UpdateProfileRequest true "Profile update payload"
// @Success      200  {object}  response.Response{data=user.PublicUser}
// @Failure      400  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Router       /user/profile [patch]
func (h *Handler) UpdateProfile(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var input UpdateProfileRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid json payload")
		return
	}

	updatedUser, err := h.svc.UpdateProfile(c.Request.Context(), userID, input)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Profile updated successfully", updatedUser)
}

// ChangePassword godoc
// @Summary      Change user password
// @Description  Updates current user password and invalidates active refresh tokens
// @Tags         user
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body user.ChangePasswordRequest true "Password change payload"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Router       /user/change-password [patch]
func (h *Handler) ChangePassword(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var input ChangePasswordRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid json payload")
		return
	}

	if err := h.svc.ChangePassword(c.Request.Context(), userID, input); err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Password changed successfully", nil)
}
