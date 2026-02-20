package mongo

import (
	"context"
	"errors"
	"time"

	"github.com/sohamify/rms-backend/internal/domain/entities"
	"github.com/sohamify/rms-backend/internal/domain/repositories"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type userRepo struct {
	coll *mongo.Collection
}

func NewUserRepository() repositories.UserRepository {
	return &userRepo{coll: GetDatabase().Collection("users")}
}

// FindByEmail returns user by email (soft-delete aware)
func (r *userRepo) FindByEmail(ctx context.Context, email string) (*entities.User, error) {
	var user entities.User
	filter := bson.M{"email": email, "deleted_at": nil}
	err := r.coll.FindOne(ctx, filter).Decode(&user)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// UpdateLastLogin updates last_login_at and updated_at
func (r *userRepo) UpdateLastLogin(ctx context.Context, id string, lastLogin time.Time) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	update := bson.M{
		"$set": bson.M{
			"last_login_at": lastLogin,
			"updated_at":    time.Now(),
		},
	}

	_, err = r.coll.UpdateOne(ctx, bson.M{"_id": objID}, update)
	return err
}

// FindByID returns user by ID (soft-delete aware)
func (r *userRepo) FindByID(ctx context.Context, id primitive.ObjectID) (*entities.User, error) {
	var user entities.User
	filter := bson.M{"_id": id, "deleted_at": nil}
	err := r.coll.FindOne(ctx, filter).Decode(&user)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindAll returns paginated users with total count (soft-delete aware)
func (r *userRepo) FindAll(ctx context.Context, filter bson.M, opts *options.FindOptions) ([]*entities.User, int64, error) {
	filter["deleted_at"] = nil // exclude soft-deleted

	cursor, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var users []*entities.User
	if err = cursor.All(ctx, &users); err != nil {
		return nil, 0, err
	}

	count, err := r.coll.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	return users, count, nil
}

// Create inserts a new user and populates its ID
func (r *userRepo) Create(ctx context.Context, user *entities.User) error {
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	res, err := r.coll.InsertOne(ctx, user)
	if err != nil {
		return err
	}

	insertedID, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return errors.New("inserted ID is not primitive.ObjectID")
	}

	user.ID = insertedID
	return nil
}

// Update updates fields and always sets updated_at
func (r *userRepo) Update(ctx context.Context, id primitive.ObjectID, fields bson.M) error {
	setFields := bson.M{
		"updated_at": time.Now(),
	}

	// Merge caller-provided fields
	for key, value := range fields {
		setFields[key] = value
	}

	update := bson.M{
		"$set": setFields,
	}

	_, err := r.coll.UpdateOne(ctx, bson.M{"_id": id}, update)
	return err
}

// SoftDelete sets deleted_at and updated_at
func (r *userRepo) SoftDelete(ctx context.Context, id primitive.ObjectID) error {
	update := bson.M{
		"$set": bson.M{
			"deleted_at": time.Now(),
			"updated_at": time.Now(),
		},
	}

	_, err := r.coll.UpdateOne(ctx, bson.M{"_id": id}, update)
	return err
}
