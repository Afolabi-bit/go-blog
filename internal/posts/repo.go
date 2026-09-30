package posts

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Repo struct {
	coll *mongo.Collection
}

func NewRepo(db *mongo.Database) *Repo {
	return &Repo{
		coll: db.Collection("posts"),
	}
}

func (r *Repo) CreatePost(ctx context.Context, post Post) (Post, error) {
	res, err := r.coll.InsertOne(ctx, post)

	if err != nil {
		return Post{}, fmt.Errorf("failed to insert post: %w", err)
	}

	id, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return Post{}, fmt.Errorf("post ID is not an ObjectID")
	}

	post.ID = id

	return post, nil
}

func (r *Repo) ListPublished(ctx context.Context, nextCursor string, limit int64, filter PostFilter) ([]Post, error) {

	query := bson.M{}

	query["status"] = StatusPublished

	if nextCursor != "" {
		objID, err := primitive.ObjectIDFromHex(nextCursor)

		if err != nil {
			return []Post{}, fmt.Errorf("Invalid parameter for next cursor: %w", err)
		}

		query["_id"] = bson.M{"$lt": objID}
	}

	if strings.TrimSpace(filter.AuthorID) != "" {
		if authorObjID, err := primitive.ObjectIDFromHex(strings.TrimSpace(filter.AuthorID)); err == nil {
			query["author_id"] = authorObjID
		}
	}

	if strings.TrimSpace(filter.Search) != "" {
		searchStr := strings.TrimSpace(filter.Search)
		query["$or"] = []bson.M{
			{"title": bson.M{"$regex": searchStr, "$options": "i"}},
			{"content": bson.M{"$regex": searchStr, "$options": "i"}},
			{"tags": bson.M{"$regex": searchStr, "$options": "i"}},
		}
	}

	if strings.TrimSpace(filter.Tag) != "" {
		query["tags"] = strings.TrimSpace(filter.Tag)
	}

	opts := options.Find()

	opts.SetSort(bson.D{{Key: "_id", Value: -1}})

	opts.SetLimit(limit)

	inCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	cursor, err := r.coll.Find(inCtx, query, opts)

	if err != nil {
		return []Post{}, fmt.Errorf("Failed to find posts: %w", err)
	}
	defer cursor.Close(inCtx)

	var posts []Post
	if err := cursor.All(inCtx, &posts); err != nil {
		return []Post{}, fmt.Errorf("Failed to decode posts: %w", err)
	}

	if posts == nil {
		posts = []Post{}
	}

	return posts, nil
}

func (r *Repo) ListByAuthor(ctx context.Context, authorID primitive.ObjectID, nextCursor string, limit int64) ([]Post, error) {
	query := bson.M{}

	query["author_id"] = authorID

	if nextCursor != "" {
		objID, err := primitive.ObjectIDFromHex(nextCursor)

		if err != nil {
			return []Post{}, fmt.Errorf("Invalid value for next cursor: %w", err)
		}

		query["_id"] = bson.M{"$lt": objID}
	}

	opts := options.Find()

	opts.SetLimit(limit)
	opts.SetSort(bson.D{{Key: "_id", Value: -1}})

	inCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	cursor, err := r.coll.Find(inCtx, query, opts)
	if err != nil {
		return []Post{}, fmt.Errorf("Failed to find posts: %w", err)
	}

	defer cursor.Close(inCtx)

	var posts []Post

	if err := cursor.All(inCtx, &posts); err != nil {
		return []Post{}, fmt.Errorf("Failed to decode posts: %w", err)
	}

	if posts == nil {
		posts = []Post{}
	}

	return posts, nil
}

func (r *Repo) ListAllAdmin(ctx context.Context, nextCursor string, maxLimit int64, filter PostFilter) ([]Post, error) {
	query := bson.M{}

	if strings.TrimSpace(filter.Status) != "" {
		query["status"] = strings.TrimSpace(filter.Status)
	}

	if strings.TrimSpace(filter.AuthorID) != "" {
		if authorObjID, err := primitive.ObjectIDFromHex(strings.TrimSpace(filter.AuthorID)); err == nil {
			query["author_id"] = authorObjID
		}
	}

	if strings.TrimSpace(filter.Search) != "" {
		searchStr := strings.TrimSpace(filter.Search)
		query["$or"] = []bson.M{
			{"title": bson.M{"$regex": searchStr, "$options": "i"}},
			{"content": bson.M{"$regex": searchStr, "$options": "i"}},
			{"tags": bson.M{"$regex": searchStr, "$options": "i"}},
		}
	}

	if strings.TrimSpace(filter.Tag) != "" {
		query["tags"] = strings.TrimSpace(filter.Tag)
	}

	if nextCursor != "" {
		objId, err := primitive.ObjectIDFromHex(nextCursor)

		if err != nil {
			return []Post{}, fmt.Errorf("Invalid value for next cursor: %w", err)
		}

		query["_id"] = bson.M{"$lt": objId}
	}

	opts := options.Find()

	opts.SetLimit(maxLimit)
	opts.SetSort(bson.D{{Key: "_id", Value: -1}})

	inCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	cursor, err := r.coll.Find(inCtx, query, opts)
	if err != nil {
		return []Post{}, fmt.Errorf("Failed to find posts: %w", err)
	}
	defer cursor.Close(inCtx)

	var posts []Post

	if err := cursor.All(inCtx, &posts); err != nil {
		return []Post{}, fmt.Errorf("Failed to decode posts: %w", err)
	}

	if posts == nil {
		return []Post{}, nil
	}

	return posts, nil
}

func (r *Repo) GetByID(ctx context.Context, postID primitive.ObjectID) (Post, error) {
	query := bson.M{"_id": postID}

	inCtx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	var post Post

	err := r.coll.FindOne(inCtx, query).Decode(&post)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return Post{}, mongo.ErrNoDocuments
		}

		return Post{}, fmt.Errorf("failed to find post: %w", err)
	}

	return post, nil
}

func (r *Repo) GetBySlug(ctx context.Context, slug string) (Post, error) {
	query := bson.M{
		"$or": []bson.M{
			{"slug": slug},
			{"previous_slugs": slug},
		},
	}

	inCtx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	var post Post

	err := r.coll.FindOne(inCtx, query).Decode(&post)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return Post{}, mongo.ErrNoDocuments
		}
		return Post{}, fmt.Errorf("failed to find post by slug: %w", err)
	}

	return post, nil
}

func (r *Repo) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	query := bson.M{"slug": slug}

	inCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	count, err := r.coll.CountDocuments(inCtx, query)
	if err != nil {
		return false, fmt.Errorf("failed to check slug existence: %w", err)
	}

	return count > 0, nil
}

func (r *Repo) Update(ctx context.Context, postID primitive.ObjectID, authorID *primitive.ObjectID, update UpdatePostRequest) (Post, error) {
	query := bson.M{"_id": postID}

	if authorID != nil {
		query["author_id"] = *authorID
	}

	fields := bson.M{"updated_at": time.Now()}

	if update.Title != nil && strings.TrimSpace(*update.Title) != "" {
		fields["title"] = strings.TrimSpace(*update.Title)
	}
	if update.Content != nil && strings.TrimSpace(*update.Content) != "" {
		fields["content"] = *update.Content
	}
	if update.Status != nil && strings.TrimSpace(*update.Status) != "" {
		fields["status"] = *update.Status
	}
	if update.Tags != nil {
		fields["tags"] = *update.Tags
	}
	if update.CoverImage != nil {
		fields["cover_image"] = strings.TrimSpace(*update.CoverImage)
	}

	updatedFields := bson.M{
		"$set": fields,
	}

	//If only "updated_at" exists in fields, nothing was actually modified
	if len(fields) == 1 {
		return r.GetByID(ctx, postID)
	}

	inCtx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	var post Post

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	err := r.coll.FindOneAndUpdate(inCtx, query, updatedFields, opts).Decode(&post)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return Post{}, mongo.ErrNoDocuments
		}

		return Post{}, fmt.Errorf("Failed to update post: %w", err)
	}

	return post, nil
}

func (r *Repo) Delete(ctx context.Context, postID primitive.ObjectID, authorID *primitive.ObjectID) error {
	query := bson.M{"_id": postID}

	if authorID != nil {
		query["author_id"] = *authorID
	}

	inCtx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	result, err := r.coll.DeleteOne(inCtx, query)

	if err != nil {
		return fmt.Errorf("Failed to delete post: %w", err)
	}

	if result.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return nil
}

func (r *Repo) IncrementCommentsCount(ctx context.Context, postID primitive.ObjectID, delta int64) error {
	inCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{"_id": postID}
	update := bson.M{
		"$inc": bson.M{"comments_count": delta},
	}

	result, err := r.coll.UpdateOne(inCtx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to increment comments count: %w", err)
	}

	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return nil
}

func (r *Repo) IncrementLikesCount(ctx context.Context, postID primitive.ObjectID, delta int64) error {
	inCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{"_id": postID}
	update := bson.M{
		"$inc": bson.M{"likes_count": delta},
	}

	result, err := r.coll.UpdateOne(inCtx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to increment likes count: %w", err)
	}

	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return nil
}

func (r *Repo) CountPublishedByAuthor(ctx context.Context, authorID primitive.ObjectID) (int64, error) {
	inCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return r.coll.CountDocuments(inCtx, bson.M{
		"author_id": authorID,
		"status":    StatusPublished,
	})
}

func (r *Repo) GetTags(ctx context.Context) ([]TagItem, error) {
	inCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"status": StatusPublished}}},
		{{Key: "$unwind", Value: "$tags"}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$tags"},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
		{{Key: "$sort", Value: bson.D{
			{Key: "count", Value: -1},
			{Key: "_id", Value: 1},
		}}},
		{{Key: "$project", Value: bson.D{
			{Key: "name", Value: "$_id"},
			{Key: "count", Value: 1},
			{Key: "_id", Value: 0},
		}}},
	}

	cursor, err := r.coll.Aggregate(inCtx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate tags: %w", err)
	}
	defer cursor.Close(inCtx)

	var tags []TagItem
	if err := cursor.All(inCtx, &tags); err != nil {
		return nil, fmt.Errorf("failed to decode tags: %w", err)
	}
	if tags == nil {
		tags = []TagItem{}
	}
	return tags, nil
}

func (r *Repo) GetAuthorStats(ctx context.Context, authorID primitive.ObjectID) (AuthorStats, error) {
	inCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"author_id": authorID}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: nil},
			{Key: "total_posts", Value: bson.D{{Key: "$sum", Value: 1}}},
			{Key: "published_posts", Value: bson.D{{Key: "$sum", Value: bson.D{
				{Key: "$cond", Value: bson.A{bson.M{"$eq": bson.A{"$status", StatusPublished}}, 1, 0}},
			}}}},
			{Key: "draft_posts", Value: bson.D{{Key: "$sum", Value: bson.D{
				{Key: "$cond", Value: bson.A{bson.M{"$eq": bson.A{"$status", StatusDraft}}, 1, 0}},
			}}}},
			{Key: "total_likes", Value: bson.D{{Key: "$sum", Value: "$likes_count"}}},
			{Key: "total_comments", Value: bson.D{{Key: "$sum", Value: "$comments_count"}}},
		}}},
	}

	cursor, err := r.coll.Aggregate(inCtx, pipeline)
	if err != nil {
		return AuthorStats{}, fmt.Errorf("failed to aggregate author stats: %w", err)
	}
	defer cursor.Close(inCtx)

	var stats []AuthorStats
	if err := cursor.All(inCtx, &stats); err != nil {
		return AuthorStats{}, fmt.Errorf("failed to decode author stats: %w", err)
	}

	if len(stats) == 0 {
		return AuthorStats{}, nil
	}
	return stats[0], nil
}

func (r *Repo) SetFeatured(ctx context.Context, postID primitive.ObjectID, isFeatured bool) (Post, error) {
	inCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if isFeatured {
		// Clear featured flag on any previously featured post
		_, _ = r.coll.UpdateMany(inCtx, bson.M{"is_featured": true}, bson.M{"$set": bson.M{"is_featured": false}})
	}

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var post Post
	err := r.coll.FindOneAndUpdate(inCtx,
		bson.M{"_id": postID},
		bson.M{"$set": bson.M{"is_featured": isFeatured, "updated_at": time.Now()}},
		opts,
	).Decode(&post)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return Post{}, mongo.ErrNoDocuments
		}
		return Post{}, fmt.Errorf("failed to set featured post: %w", err)
	}

	return post, nil
}

func (r *Repo) GetFeaturedPost(ctx context.Context) (Post, error) {
	inCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var post Post
	err := r.coll.FindOne(inCtx, bson.M{
		"is_featured": true,
		"status":      StatusPublished,
	}).Decode(&post)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return Post{}, mongo.ErrNoDocuments
		}
		return Post{}, fmt.Errorf("failed to find featured post: %w", err)
	}

	return post, nil
}


