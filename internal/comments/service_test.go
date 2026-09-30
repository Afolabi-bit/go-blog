package comments_test

import (
	"blog-api/internal/comments"
	"blog-api/internal/posts"
	"blog-api/internal/user"
	"context"
	"errors"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type mockCommentRepo struct {
	createFunc            func(ctx context.Context, c comments.Comment) (comments.Comment, error)
	findByIDFunc          func(ctx context.Context, id primitive.ObjectID) (comments.Comment, error)
	listByPostIDFunc      func(ctx context.Context, postID primitive.ObjectID, nextCursor string, limit int64) ([]comments.Comment, error)
	listRootCommentsFunc  func(ctx context.Context, postID primitive.ObjectID, nextCursor string, limit int64) ([]comments.Comment, error)
	listRepliesFunc       func(ctx context.Context, postID primitive.ObjectID, parentIDs []primitive.ObjectID) ([]comments.Comment, error)
	listAllAdminFunc      func(ctx context.Context, nextCursor string, limit int64, filter comments.AdminCommentFilter) ([]comments.Comment, error)
	deleteFunc            func(ctx context.Context, id primitive.ObjectID) error
	deleteRepliesByParent func(ctx context.Context, parentID primitive.ObjectID) (int64, error)
}

func (m *mockCommentRepo) Create(ctx context.Context, c comments.Comment) (comments.Comment, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, c)
	}
	c.ID = primitive.NewObjectID()
	return c, nil
}

func (m *mockCommentRepo) FindByID(ctx context.Context, id primitive.ObjectID) (comments.Comment, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return comments.Comment{}, mongo.ErrNoDocuments
}

func (m *mockCommentRepo) ListByPostID(ctx context.Context, postID primitive.ObjectID, nextCursor string, limit int64) ([]comments.Comment, error) {
	if m.listByPostIDFunc != nil {
		return m.listByPostIDFunc(ctx, postID, nextCursor, limit)
	}
	return []comments.Comment{}, nil
}

func (m *mockCommentRepo) ListRootComments(ctx context.Context, postID primitive.ObjectID, nextCursor string, limit int64) ([]comments.Comment, error) {
	if m.listRootCommentsFunc != nil {
		return m.listRootCommentsFunc(ctx, postID, nextCursor, limit)
	}
	return []comments.Comment{}, nil
}

func (m *mockCommentRepo) ListReplies(ctx context.Context, postID primitive.ObjectID, parentIDs []primitive.ObjectID) ([]comments.Comment, error) {
	if m.listRepliesFunc != nil {
		return m.listRepliesFunc(ctx, postID, parentIDs)
	}
	return []comments.Comment{}, nil
}

func (m *mockCommentRepo) ListAllAdmin(ctx context.Context, nextCursor string, limit int64, filter comments.AdminCommentFilter) ([]comments.Comment, error) {
	if m.listAllAdminFunc != nil {
		return m.listAllAdminFunc(ctx, nextCursor, limit, filter)
	}
	return []comments.Comment{}, nil
}

func (m *mockCommentRepo) Delete(ctx context.Context, id primitive.ObjectID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func (m *mockCommentRepo) DeleteRepliesByParentID(ctx context.Context, parentID primitive.ObjectID) (int64, error) {
	if m.deleteRepliesByParent != nil {
		return m.deleteRepliesByParent(ctx, parentID)
	}
	return 0, nil
}

type mockPostRepo struct {
	getByIDFunc           func(ctx context.Context, postID primitive.ObjectID) (posts.Post, error)
	incrementCommentsFunc func(ctx context.Context, postID primitive.ObjectID, delta int64) error
}

func (m *mockPostRepo) GetByID(ctx context.Context, postID primitive.ObjectID) (posts.Post, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, postID)
	}
	return posts.Post{
		ID:     postID,
		Status: posts.StatusPublished,
	}, nil
}

func (m *mockPostRepo) IncrementCommentsCount(ctx context.Context, postID primitive.ObjectID, delta int64) error {
	if m.incrementCommentsFunc != nil {
		return m.incrementCommentsFunc(ctx, postID, delta)
	}
	return nil
}

type mockUserRepo struct {
	findUserByIDFunc func(ctx context.Context, id string) (user.User, error)
}

func (m *mockUserRepo) FinduserByID(ctx context.Context, id string) (user.User, error) {
	if m.findUserByIDFunc != nil {
		return m.findUserByIDFunc(ctx, id)
	}
	objID, _ := primitive.ObjectIDFromHex(id)
	return user.User{
		ID:        objID,
		FirstName: "Jane",
		LastName:  "Reader",
		Email:     "jane@reader.com",
	}, nil
}

func TestAddComment_Success(t *testing.T) {
	postID := primitive.NewObjectID()
	authorID := primitive.NewObjectID()

	commentRepo := &mockCommentRepo{}
	postRepo := &mockPostRepo{}
	userRepo := &mockUserRepo{}

	svc := comments.NewService(commentRepo, postRepo, userRepo)

	input := comments.CreateCommentRequest{
		Content: "Great article!",
	}

	result, err := svc.AddComment(context.Background(), postID.Hex(), authorID.Hex(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Content != "Great article!" {
		t.Errorf("expected content 'Great article!', got '%s'", result.Content)
	}
	if result.AuthorName != "Jane Reader" {
		t.Errorf("expected author name 'Jane Reader', got '%s'", result.AuthorName)
	}
}

func TestAddComment_UnpublishedPost(t *testing.T) {
	postID := primitive.NewObjectID()
	authorID := primitive.NewObjectID()

	postRepo := &mockPostRepo{
		getByIDFunc: func(ctx context.Context, id primitive.ObjectID) (posts.Post, error) {
			return posts.Post{
				ID:     id,
				Status: posts.StatusDraft,
			}, nil
		},
	}

	svc := comments.NewService(&mockCommentRepo{}, postRepo, &mockUserRepo{})

	_, err := svc.AddComment(context.Background(), postID.Hex(), authorID.Hex(), comments.CreateCommentRequest{Content: "Nice draft"})
	if !errors.Is(err, comments.ErrPostNotFound) {
		t.Fatalf("expected ErrPostNotFound, got: %v", err)
	}
}

func TestAddComment_NestedReplySuccess(t *testing.T) {
	postID := primitive.NewObjectID()
	parentID := primitive.NewObjectID()
	authorID := primitive.NewObjectID()

	commentRepo := &mockCommentRepo{
		findByIDFunc: func(ctx context.Context, id primitive.ObjectID) (comments.Comment, error) {
			return comments.Comment{
				ID:     parentID,
				PostID: postID,
			}, nil
		},
	}

	svc := comments.NewService(commentRepo, &mockPostRepo{}, &mockUserRepo{})

	parentHex := parentID.Hex()
	input := comments.CreateCommentRequest{
		Content:  "I agree with your comment!",
		ParentID: &parentHex,
	}

	res, err := svc.AddComment(context.Background(), postID.Hex(), authorID.Hex(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.ParentID == nil || *res.ParentID != parentID {
		t.Errorf("expected ParentID to match %s", parentHex)
	}
}

func TestDeleteComment_Authorization(t *testing.T) {
	commentID := primitive.NewObjectID()
	postID := primitive.NewObjectID()
	commentAuthorID := primitive.NewObjectID()
	postAuthorID := primitive.NewObjectID()
	randomUserID := primitive.NewObjectID()

	existingComment := comments.Comment{
		ID:        commentID,
		PostID:    postID,
		AuthorID:  commentAuthorID,
		Content:   "A comment",
		CreatedAt: time.Now(),
	}

	postRepo := &mockPostRepo{
		getByIDFunc: func(ctx context.Context, id primitive.ObjectID) (posts.Post, error) {
			return posts.Post{
				ID:       postID,
				AuthorID: postAuthorID,
			}, nil
		},
	}

	tests := []struct {
		name        string
		requesterID primitive.ObjectID
		role        string
		expectErr   bool
	}{
		{"Comment author can delete", commentAuthorID, user.RoleReader, false},
		{"Post author can delete", postAuthorID, user.RoleAuthor, false},
		{"Admin can delete", randomUserID, user.RoleAdmin, false},
		{"Unrelated reader cannot delete", randomUserID, user.RoleReader, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commentRepo := &mockCommentRepo{
				findByIDFunc: func(ctx context.Context, id primitive.ObjectID) (comments.Comment, error) {
					return existingComment, nil
				},
			}

			svc := comments.NewService(commentRepo, postRepo, &mockUserRepo{})

			err := svc.DeleteComment(context.Background(), commentID.Hex(), tt.requesterID.Hex(), tt.role)
			if tt.expectErr && !errors.Is(err, comments.ErrForbidden) {
				t.Fatalf("expected ErrForbidden, got: %v", err)
			}
			if !tt.expectErr && err != nil {
				t.Fatalf("expected success, got: %v", err)
			}
		})
	}
}

func TestListComments_NestedReplies(t *testing.T) {
	postID := primitive.NewObjectID()
	root1ID := primitive.NewObjectID()
	reply1ID := primitive.NewObjectID()

	rootComment := comments.Comment{
		ID:         root1ID,
		PostID:     postID,
		Content:    "Root comment",
		CreatedAt:  time.Now(),
		AuthorName: "Alice",
	}

	replyComment := comments.Comment{
		ID:         reply1ID,
		PostID:     postID,
		ParentID:   &root1ID,
		Content:    "Reply to root",
		CreatedAt:  time.Now().Add(time.Minute),
		AuthorName: "Bob",
	}

	repo := &mockCommentRepo{
		listRootCommentsFunc: func(ctx context.Context, pid primitive.ObjectID, cursor string, limit int64) ([]comments.Comment, error) {
			return []comments.Comment{rootComment}, nil
		},
		listRepliesFunc: func(ctx context.Context, pid primitive.ObjectID, parentIDs []primitive.ObjectID) ([]comments.Comment, error) {
			return []comments.Comment{replyComment}, nil
		},
	}

	svc := comments.NewService(repo, &mockPostRepo{}, &mockUserRepo{})

	tree, meta, err := svc.ListComments(context.Background(), postID.Hex(), "", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tree) != 1 {
		t.Fatalf("expected 1 root comment, got %d", len(tree))
	}
	if len(tree[0].Replies) != 1 {
		t.Fatalf("expected 1 nested reply, got %d", len(tree[0].Replies))
	}
	if tree[0].Replies[0].Content != "Reply to root" {
		t.Errorf("expected reply content 'Reply to root', got '%s'", tree[0].Replies[0].Content)
	}
	if meta.Count != 1 {
		t.Errorf("expected count 1, got %d", meta.Count)
	}
}

func TestListAllAdmin_ModerationQueue(t *testing.T) {
	postID := primitive.NewObjectID()
	cID := primitive.NewObjectID()

	repo := &mockCommentRepo{
		listAllAdminFunc: func(ctx context.Context, cursor string, limit int64, filter comments.AdminCommentFilter) ([]comments.Comment, error) {
			return []comments.Comment{
				{
					ID:      cID,
					PostID:  postID,
					Content: "Spam comment under review",
				},
			}, nil
		},
	}

	svc := comments.NewService(repo, &mockPostRepo{}, &mockUserRepo{})

	items, meta, err := svc.ListAllAdmin(context.Background(), "", 10, comments.AdminCommentFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("expected 1 item in admin queue, got %d", len(items))
	}
	if meta.Count != 1 {
		t.Errorf("expected count 1, got %d", meta.Count)
	}
}

