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

type billRepo struct {
	coll *mongo.Collection
}

func NewBillRepository() repositories.BillRepository {
	return &billRepo{coll: GetDatabase().Collection("bills")}
}

func (r *billRepo) Create(ctx context.Context, bill *entities.Bill) (*entities.Bill, error) {
	bill.CreatedAt = time.Now()
	bill.UpdatedAt = time.Now()
	bill.Status = entities.BillGenerated

	res, err := r.coll.InsertOne(ctx, bill)
	if err != nil {
		return nil, err
	}

	bill.ID = res.InsertedID.(primitive.ObjectID)
	return bill, nil
}

func (r *billRepo) FindByID(ctx context.Context, id primitive.ObjectID) (*entities.Bill, error) {
	var bill entities.Bill
	err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&bill)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &bill, err
}

func (r *billRepo) FindByOrderID(ctx context.Context, orderID primitive.ObjectID) (*entities.Bill, error) {
	var bill entities.Bill
	err := r.coll.FindOne(ctx, bson.M{"order_id": orderID}).Decode(&bill)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &bill, err
}

func (r *billRepo) Update(ctx context.Context, id primitive.ObjectID, fields bson.M) error {
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
