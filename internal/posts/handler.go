package posts

import (
	"blog-api/internal/middleware"
	"blog-api/internal/response"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
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
	case errors.Is(err, ErrInvalidInput), errors.Is(err, ErrInvalidID):
		response.Error(c, http.StatusBadRequest, err.Error())
	default:
		log.Printf("[ERROR] Posts handler unexpected error: %v", err)
		response.Error(c, http.StatusInternalServerError, "internal server error")
	}
}

// CreatePost godoc
// @Summary      Create a new post
// @Description  Creates a draft or published post (Author or Admin role required)
// @Tags         posts
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body posts.CreatePostRequest true "Post creation payload"
// @Success      201  {object}  response.Response{data=posts.Post}
// @Failure      400  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Failure      403  {object}  response.Response
// @Router       /api/posts [post]
func (h *Handler) CreatePost(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)

	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := h.svc.user.FinduserByID(c, userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	var post CreatePostRequest

	if err = c.ShouldBindJSON(&post); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid json payload")
		return
	}

	authorName := user.FirstName + " " + user.LastName
	createdPost, err := h.svc.CreateNewPost(c.Request.Context(), user.ID, authorName, post)

	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusCreated, "post successfully created", createdPost)
}

// GetPostByID godoc
// @Summary      Get a single post by ID
// @Description  Fetch post details. Published posts are public; draft posts require author or admin credentials.
// @Tags         posts
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string true "Post ID (24-char hex MongoDB ObjectID)"
// @Success      200  {object}  response.Response{data=posts.Post}
// @Failure      400  {object}  response.Response
// @Failure      403  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Router       /api/posts/{id} [get]
func (h *Handler) GetPostByID(c *gin.Context) {
	postID := c.Param("id")

	var requesterIDPtr, requesterRolePtr *string
	if userID, ok := middleware.GetUserID(c); ok {
		requesterIDPtr = &userID
	}
	if userRole, ok := middleware.GetRole(c); ok {
		requesterRolePtr = &userRole
	}
	post, err := h.svc.GetPostByID(c.Request.Context(), postID, requesterIDPtr, requesterRolePtr)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Post fetched successfully", post)
}

// GetPostBySlug godoc
// @Summary      Get a single post by slug
// @Description  Fetch post details by slug. Published posts are public; draft posts require author or admin credentials.
// @Tags         posts
// @Produce      json
// @Security     BearerAuth
// @Param        slug path      string true "Post Slug"
// @Success      200  {object}  response.Response{data=posts.Post}
// @Failure      403  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Router       /api/posts/slug/{slug} [get]
func (h *Handler) GetPostBySlug(c *gin.Context) {
	slug := c.Param("slug")

	var requesterIDPtr, requesterRolePtr *string
	if userID, ok := middleware.GetUserID(c); ok {
		requesterIDPtr = &userID
	}
	if userRole, ok := middleware.GetRole(c); ok {
		requesterRolePtr = &userRole
	}

	post, err := h.svc.GetPostBySlug(c.Request.Context(), slug, requesterIDPtr, requesterRolePtr)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Post fetched successfully", post)
}

// ListPublicPosts godoc
// @Summary      List published posts
// @Description  Browse published blog posts with search, tag filtering, and cursor pagination
// @Tags         posts
// @Produce      json
// @Param        search query string false "Search keyword in post titles"
// @Param        tag    query string false "Filter by tag"
// @Param        limit  query int    false "Number of posts to return (default 10, max 20)"
// @Param        cursor query string false "Pagination cursor"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Router       /api/posts [get]
func (h *Handler) ListPublicPosts(c *gin.Context) {
	var filter PostFilter
	err := c.ShouldBindQuery(&filter)

	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid query parameters")
		return
	}

	posts, meta, err := h.svc.ListPublicPosts(c.Request.Context(), filter.Cursor, filter.Limit, filter)

	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Fetched posts successfully", gin.H{
		"posts":      posts,
		"pagination": meta,
	})

}

// ListMyPosts godoc
// @Summary      Author post dashboard
// @Description  Lists all posts created by the authenticated author, including drafts and published posts
// @Tags         posts
// @Produce      json
// @Security     BearerAuth
// @Param        limit  query int    false "Limit (default 10, max 20)"
// @Param        cursor query string false "Pagination cursor"
// @Success      200  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Failure      403  {object}  response.Response
// @Router       /api/my-posts [get]
func (h *Handler) ListMyPosts(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	cursor := c.Query("cursor")
	limitStr := c.DefaultQuery("limit", "10")

	limit, err := strconv.ParseInt(limitStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid limit parameter")
		return
	}

	posts, meta, err := h.svc.ListMyPosts(c.Request.Context(), objID, cursor, limit)

	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Fetched posts successfully", gin.H{
		"posts":      posts,
		"pagination": meta,
	})
}

// ListAllAdmin godoc
// @Summary      Admin post moderation list
// @Description  Lists all posts across all authors and statuses (Admin role required)
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Param        search query string false "Search keyword"
// @Param        tag    query string false "Filter by tag"
// @Param        status query string false "Filter by status"
// @Param        limit  query int    false "Limit"
// @Param        cursor query string false "Pagination cursor"
// @Success      200  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Failure      403  {object}  response.Response
// @Router       /api/admin/posts [get]
func (h *Handler) ListAllAdmin(c *gin.Context) {
	var filter PostFilter
	err := c.ShouldBindQuery(&filter)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid query parameters")
		return
	}

	posts, meta, err := h.svc.ListAllAdmin(c.Request.Context(), filter.Cursor, filter.Limit, filter)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Fetched posts successfully", gin.H{
		"posts":      posts,
		"pagination": meta,
	})
}

// UpdatePost godoc
// @Summary      Update a post
// @Description  Partially update a post's title, content, or status. Authors can only update their own posts; Admins can update any post.
// @Tags         posts
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id      path string                  true "Post ID (24-char hex MongoDB ObjectID)"
// @Param        request body posts.UpdatePostRequest true "Update payload"
// @Success      200     {object} response.Response{data=posts.Post}
// @Failure      400     {object} response.Response
// @Failure      401     {object} response.Response
// @Failure      403     {object} response.Response
// @Failure      404     {object} response.Response
// @Router       /api/posts/{id} [patch]
func (h *Handler) UpdatePost(c *gin.Context) {
	postID := c.Param("id")

	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	userRole, ok := middleware.GetRole(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	userObjID, err := primitive.ObjectIDFromHex(userID)

	if err != nil {
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var update UpdatePostRequest

	err = c.ShouldBindJSON(&update)

	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid json payload")
		return
	}

	post, err := h.svc.UpdatePost(c.Request.Context(), postID, userObjID, userRole, update)

	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Post updated successfully", post)
}

// DeletePost godoc
// @Summary      Delete a post
// @Description  Deletes a post. Authors can only delete their own posts; Admins can delete any post.
// @Tags         posts
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string true "Post ID (24-char hex MongoDB ObjectID)"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Failure      403  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Router       /api/posts/{id} [delete]
func (h *Handler) DeletePost(c *gin.Context) {
	postID := c.Param("id")

	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	userRole, ok := middleware.GetRole(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	userObjID, err := primitive.ObjectIDFromHex(userID)

	if err != nil {
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	err = h.svc.DeletePost(c.Request.Context(), postID, userObjID, userRole)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Post successfully deleted", nil)
}
