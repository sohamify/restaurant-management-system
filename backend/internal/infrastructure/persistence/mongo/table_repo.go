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

type tableRepo struct {
	coll *mongo.Collection
}

func NewTableRepository() repositories.TableRepository {
	return &tableRepo{coll: GetDatabase().Collection("tables")}
}

func (r *tableRepo) Create(ctx context.Context, table *entities.Table) (*entities.Table, error) {
	table.CreatedAt = time.Now()
	table.UpdatedAt = time.Now()
	table.Status = entities.TableAvailable

	res, err := r.coll.InsertOne(ctx, table)
	if err != nil {
		return nil, err
	}

	insertedID, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return nil, errors.New("inserted ID is not ObjectID")
	}

	table.ID = insertedID
	return table, nil
}

func (r *tableRepo) FindByID(ctx context.Context, id primitive.ObjectID) (*entities.Table, error) {
	var table entities.Table
	err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&table)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &table, err
}

func (r *tableRepo) FindByTableNumber(ctx context.Context, number int) (*entities.Table, error) {
	var table entities.Table
	err := r.coll.FindOne(ctx, bson.M{"table_number": number}).Decode(&table)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &table, err
}

func (r *tableRepo) FindAll(ctx context.Context, filter bson.M, opts *options.FindOptions) ([]*entities.Table, int64, error) {
	cursor, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var tables []*entities.Table
	if err = cursor.All(ctx, &tables); err != nil {
		return nil, 0, err
	}

	count, err := r.coll.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	return tables, count, nil
}

func (r *tableRepo) FindAssignedToWaiter(ctx context.Context, waiterID primitive.ObjectID) ([]*entities.Table, error) {
	filter := bson.M{
		"assigned_waiter_id": waiterID,
		"deleted_at":         nil,
	}
	cursor, err := r.coll.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var tables []*entities.Table
	if err = cursor.All(ctx, &tables); err != nil {
		return nil, err
	}

	return tables, nil
}

func (r *tableRepo) Update(ctx context.Context, id primitive.ObjectID, fields bson.M) error {
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

func (r *tableRepo) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.coll.DeleteOne(ctx, bson.M{"_id": id})
	return err
}
