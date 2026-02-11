package mongo

import (
	"context"
	"time"

	"github.com/sohamify/rms-backend/internal/domain/entities"
	"github.com/sohamify/rms-backend/internal/domain/repositories"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type userRepo struct {
	coll *mongo.Collection
}

func NewUserRepository() repositories.UserRepository {
	return &userRepo{coll: GetDatabase().Collection("users")}
}

func (r *userRepo) FindByEmail(ctx context.Context, email string) (*entities.User, error) {
	var user entities.User
	filter := bson.M{"email": email, "deleted_at": nil} // Soft delete filter
	err := r.coll.FindOne(ctx, filter).Decode(&user)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &user, err
}

func (r *userRepo) UpdateLastLogin(ctx context.Context, id string, lastLogin time.Time) error {
	objID, _ := primitive.ObjectIDFromHex(id)
	filter := bson.M{"_id": objID}
	update := bson.M{"$set": bson.M{"last_login_at": lastLogin, "updated_at": time.Now()}}
	_, err := r.coll.UpdateOne(ctx, filter, update)
	return err
}
