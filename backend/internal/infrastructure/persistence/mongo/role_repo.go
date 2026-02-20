package mongo

import (
	"context"
	"time"

	"github.com/sohamify/rms-backend/internal/domain/entities"
	"github.com/sohamify/rms-backend/internal/domain/repositories"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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

func (r *roleRepo) FindByName(ctx context.Context, name string) (*entities.Role, error) {
	var role entities.Role
	err := r.coll.FindOne(ctx, bson.M{"name": name}).Decode(&role)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &role, err
}

func (r *roleRepo) Create(ctx context.Context, role *entities.Role) error {
	role.CreatedAt = time.Now()
	_, err := r.coll.InsertOne(ctx, role)
	return err
}

func (r *roleRepo) Update(ctx context.Context, id primitive.ObjectID, fields bson.M) error {
	setFields := bson.M{
		"updated_at": time.Now(),
	}

	for k, v := range fields {
		setFields[k] = v
	}

	update := bson.M{"$set": setFields}

	_, err := r.coll.UpdateOne(ctx, bson.M{"_id": id}, update)
	return err
}

func (r *roleRepo) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.coll.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func (r *roleRepo) ListAll(ctx context.Context, filter bson.M, opts *options.FindOptions) ([]*entities.Role, int64, error) {
	cursor, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var roles []*entities.Role
	if err := cursor.All(ctx, &roles); err != nil {
		return nil, 0, err
	}

	count, err := r.coll.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	return roles, count, nil
}
