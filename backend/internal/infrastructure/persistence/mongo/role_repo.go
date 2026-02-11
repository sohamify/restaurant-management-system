package mongo

import (
	"context"

	"github.com/sohamify/rms-backend/internal/domain/entities"
	"github.com/sohamify/rms-backend/internal/domain/repositories"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type roleRepo struct {
	coll *mongo.Collection
}

func NewRoleRepository() repositories.RoleRepository {
	return &roleRepo{coll: GetDatabase().Collection("roles")}
}

func (r *roleRepo) FindByID(ctx context.Context, id primitive.ObjectID) (*entities.Role, error) {
	var role entities.Role
	err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&role)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &role, err
}
