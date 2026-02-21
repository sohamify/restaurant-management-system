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

type paymentRepo struct {
	coll *mongo.Collection
}

func NewPaymentRepository() repositories.PaymentRepository {
	return &paymentRepo{coll: GetDatabase().Collection("payments")}
}

func (r *paymentRepo) Create(ctx context.Context, payment *entities.Payment) (*entities.Payment, error) {
	payment.CreatedAt = time.Now()
	payment.UpdatedAt = time.Now()

	res, err := r.coll.InsertOne(ctx, payment)
	if err != nil {
		return nil, err
	}

	payment.ID = res.InsertedID.(primitive.ObjectID)
	return payment, nil
}

func (r *paymentRepo) FindByID(ctx context.Context, id primitive.ObjectID) (*entities.Payment, error) {
	var payment entities.Payment
	err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&payment)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &payment, err
}

func (r *paymentRepo) FindByOrderID(ctx context.Context, orderID primitive.ObjectID) ([]*entities.Payment, error) {
	cursor, err := r.coll.Find(ctx, bson.M{"order_id": orderID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var payments []*entities.Payment
	if err = cursor.All(ctx, &payments); err != nil {
		return nil, err
	}

	return payments, nil
}

func (r *paymentRepo) Update(ctx context.Context, id primitive.ObjectID, fields bson.M) error {
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

func (r *paymentRepo) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.coll.DeleteOne(ctx, bson.M{"_id": id})
	return err
}
