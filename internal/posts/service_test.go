package posts_test

import (
	"blog-api/internal/posts"
	"blog-api/internal/user"
	"context"
	"errors"
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type mockPostRepo struct {
	createPostFunc func(ctx context.Context, post posts.Post) (posts.Post, error)
	getByIDFunc    func(ctx context.Context, postID primitive.ObjectID) (posts.Post, error)
	getBySlugFunc  func(ctx context.Context, slug string) (posts.Post, error)
	existsBySlug   func(ctx context.Context, slug string) (bool, error)
	listPublished  func(ctx context.Context, nextCursor string, limit int64, filter posts.PostFilter) ([]posts.Post, error)
	listByAuthor   func(ctx context.Context, authorID primitive.ObjectID, nextCursor string, limit int64) ([]posts.Post, error)
	listAllAdmin   func(ctx context.Context, nextCursor string, maxLimit int64, filter posts.PostFilter) ([]posts.Post, error)
	updateFunc     func(ctx context.Context, postID primitive.ObjectID, authorID *primitive.ObjectID, update posts.UpdatePostRequest) (posts.Post, error)
	deleteFunc     func(ctx context.Context, postID primitive.ObjectID, authorID *primitive.ObjectID) error
	incrComments   func(ctx context.Context, postID primitive.ObjectID, delta int64) error
	incrLikes      func(ctx context.Context, postID primitive.ObjectID, delta int64) error
}

func (m *mockPostRepo) CreatePost(ctx context.Context, post posts.Post) (posts.Post, error) {
	if m.createPostFunc != nil {
		return m.createPostFunc(ctx, post)
	}
	post.ID = primitive.NewObjectID()
	return post, nil
}

func (m *mockPostRepo) GetByID(ctx context.Context, postID primitive.ObjectID) (posts.Post, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, postID)
	}
	return posts.Post{ID: postID, Status: posts.StatusPublished}, nil
}

func (m *mockPostRepo) GetBySlug(ctx context.Context, slug string) (posts.Post, error) {
	if m.getBySlugFunc != nil {
		return m.getBySlugFunc(ctx, slug)
	}
	return posts.Post{ID: primitive.NewObjectID(), Slug: slug, Status: posts.StatusPublished}, nil
}

func (m *mockPostRepo) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	if m.existsBySlug != nil {
		return m.existsBySlug(ctx, slug)
	}
	return false, nil
}

func (m *mockPostRepo) ListPublished(ctx context.Context, nextCursor string, limit int64, filter posts.PostFilter) ([]posts.Post, error) {
	if m.listPublished != nil {
		return m.listPublished(ctx, nextCursor, limit, filter)
	}
	return []posts.Post{}, nil
}

func (m *mockPostRepo) ListByAuthor(ctx context.Context, authorID primitive.ObjectID, nextCursor string, limit int64) ([]posts.Post, error) {
	if m.listByAuthor != nil {
		return m.listByAuthor(ctx, authorID, nextCursor, limit)
	}
	return []posts.Post{}, nil
}

func (m *mockPostRepo) ListAllAdmin(ctx context.Context, nextCursor string, maxLimit int64, filter posts.PostFilter) ([]posts.Post, error) {
	if m.listAllAdmin != nil {
		return m.listAllAdmin(ctx, nextCursor, maxLimit, filter)
	}
	return []posts.Post{}, nil
}

func (m *mockPostRepo) Update(ctx context.Context, postID primitive.ObjectID, authorID *primitive.ObjectID, update posts.UpdatePostRequest) (posts.Post, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, postID, authorID, update)
	}
	return posts.Post{ID: postID}, nil
}

func (m *mockPostRepo) Delete(ctx context.Context, postID primitive.ObjectID, authorID *primitive.ObjectID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, postID, authorID)
	}
	return nil
}

func (m *mockPostRepo) IncrementCommentsCount(ctx context.Context, postID primitive.ObjectID, delta int64) error {
	if m.incrComments != nil {
		return m.incrComments(ctx, postID, delta)
	}
	return nil
}

func (m *mockPostRepo) IncrementLikesCount(ctx context.Context, postID primitive.ObjectID, delta int64) error {
	if m.incrLikes != nil {
		return m.incrLikes(ctx, postID, delta)
	}
	return nil
}

type mockUserRepo struct{}

func (m *mockUserRepo) FinduserByID(ctx context.Context, id string) (user.User, error) {
	objID, _ := primitive.ObjectIDFromHex(id)
	return user.User{ID: objID, Role: user.RoleAuthor}, nil
}

func TestCreateNewPost_Success(t *testing.T) {
	authorID := primitive.NewObjectID()
	repo := &mockPostRepo{
		createPostFunc: func(ctx context.Context, post posts.Post) (posts.Post, error) {
			post.ID = primitive.NewObjectID()
			return post, nil
		},
	}

	svc := posts.NewService(repo, &mockUserRepo{})

	input := posts.CreatePostRequest{
		Title:      "My First Blog Post",
		Content:    "Hello world this is a test blog post.",
		CoverImage: "https://example.com/cover.jpg",
		Status:     posts.StatusPublished,
		Tags:       []string{"go", "tech"},
	}

	created, err := svc.CreateNewPost(context.Background(), authorID, "Jane Doe", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if created.Title != "My First Blog Post" {
		t.Errorf("expected title 'My First Blog Post', got '%s'", created.Title)
	}
	if created.Slug != "my-first-blog-post" {
		t.Errorf("expected slug 'my-first-blog-post', got '%s'", created.Slug)
	}
	if created.CoverImage != "https://example.com/cover.jpg" {
		t.Errorf("expected cover image 'https://example.com/cover.jpg', got '%s'", created.CoverImage)
	}
}

func TestCreateNewPost_SlugCollisionResolution(t *testing.T) {
	authorID := primitive.NewObjectID()
	callCount := 0
	repo := &mockPostRepo{
		existsBySlug: func(ctx context.Context, slug string) (bool, error) {
			callCount++
			if callCount == 1 {
				return true, nil // first slug "go-rocks" exists
			}
			return false, nil // "go-rocks-1" is available
		},
	}

	svc := posts.NewService(repo, &mockUserRepo{})

	input := posts.CreatePostRequest{
		Title:   "Go Rocks",
		Content: "Golang is awesome.",
	}

	created, err := svc.CreateNewPost(context.Background(), authorID, "Jane Doe", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if created.Slug != "go-rocks-1" {
		t.Errorf("expected slug 'go-rocks-1', got '%s'", created.Slug)
	}
}

func TestCreateNewPost_InvalidInput(t *testing.T) {
	authorID := primitive.NewObjectID()
	svc := posts.NewService(&mockPostRepo{}, &mockUserRepo{})

	_, err := svc.CreateNewPost(context.Background(), authorID, "Jane", posts.CreatePostRequest{
		Title:   "",
		Content: "Some content",
	})
	if !errors.Is(err, posts.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for empty title, got: %v", err)
	}
}

func TestGetPostByID_DraftPermissions(t *testing.T) {
	postID := primitive.NewObjectID()
	authorID := primitive.NewObjectID()
	otherUserID := primitive.NewObjectID().Hex()
	adminRole := user.RoleAdmin
	readerRole := user.RoleReader
	authorRole := user.RoleAuthor

	repo := &mockPostRepo{
		getByIDFunc: func(ctx context.Context, id primitive.ObjectID) (posts.Post, error) {
			return posts.Post{
				ID:       id,
				AuthorID: authorID,
				Status:   posts.StatusDraft,
			}, nil
		},
	}

	svc := posts.NewService(repo, &mockUserRepo{})

	// Unauthenticated reader cannot view draft
	_, err := svc.GetPostByID(context.Background(), postID.Hex(), nil, nil)
	if !errors.Is(err, posts.ErrNotFound) {
		t.Errorf("expected ErrNotFound for unauthenticated user viewing draft, got: %v", err)
	}

	// Another reader gets forbidden
	_, err = svc.GetPostByID(context.Background(), postID.Hex(), &otherUserID, &readerRole)
	if !errors.Is(err, posts.ErrForbidden) {
		t.Errorf("expected ErrForbidden for unrelated reader viewing draft, got: %v", err)
	}

	// Author can view own draft
	authorIDHex := authorID.Hex()
	post, err := svc.GetPostByID(context.Background(), postID.Hex(), &authorIDHex, &authorRole)
	if err != nil {
		t.Fatalf("unexpected error for author viewing draft: %v", err)
	}
	if post.ID != postID {
		t.Errorf("expected post id %s, got %s", postID.Hex(), post.ID.Hex())
	}

	// Admin can view any draft
	post, err = svc.GetPostByID(context.Background(), postID.Hex(), &otherUserID, &adminRole)
	if err != nil {
		t.Fatalf("unexpected error for admin viewing draft: %v", err)
	}
	if post.ID != postID {
		t.Errorf("expected post id %s, got %s", postID.Hex(), post.ID.Hex())
	}
}

func TestUpdatePost_Authorization(t *testing.T) {
	postID := primitive.NewObjectID()
	authorID := primitive.NewObjectID()
	otherUserID := primitive.NewObjectID()

	repo := &mockPostRepo{
		updateFunc: func(ctx context.Context, pID primitive.ObjectID, aID *primitive.ObjectID, update posts.UpdatePostRequest) (posts.Post, error) {
			if aID != nil && *aID != authorID {
				return posts.Post{}, mongo.ErrNoDocuments
			}
			return posts.Post{ID: pID, Title: "Updated Title"}, nil
		},
	}

	svc := posts.NewService(repo, &mockUserRepo{})

	newTitle := "Updated Title"
	updateReq := posts.UpdatePostRequest{Title: &newTitle}

	// Author updating own post
	updated, err := svc.UpdatePost(context.Background(), postID.Hex(), authorID, user.RoleAuthor, updateReq)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Title != "Updated Title" {
		t.Errorf("expected title 'Updated Title', got '%s'", updated.Title)
	}

	// Admin updating post
	_, err = svc.UpdatePost(context.Background(), postID.Hex(), otherUserID, user.RoleAdmin, updateReq)
	if err != nil {
		t.Fatalf("unexpected error for admin update: %v", err)
	}

	// Reader cannot update post
	_, err = svc.UpdatePost(context.Background(), postID.Hex(), otherUserID, user.RoleReader, updateReq)
	if !errors.Is(err, posts.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for reader updating post, got: %v", err)
	}
}

func TestDeletePost_Authorization(t *testing.T) {
	postID := primitive.NewObjectID()
	authorID := primitive.NewObjectID()
	otherUserID := primitive.NewObjectID()

	repo := &mockPostRepo{
		deleteFunc: func(ctx context.Context, pID primitive.ObjectID, aID *primitive.ObjectID) error {
			return nil
		},
	}

	svc := posts.NewService(repo, &mockUserRepo{})

	// Author deleting own post
	err := svc.DeletePost(context.Background(), postID.Hex(), authorID, user.RoleAuthor)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Admin deleting post
	err = svc.DeletePost(context.Background(), postID.Hex(), otherUserID, user.RoleAdmin)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Reader forbidden
	err = svc.DeletePost(context.Background(), postID.Hex(), otherUserID, user.RoleReader)
	if !errors.Is(err, posts.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for reader deleting post, got: %v", err)
	}
}
