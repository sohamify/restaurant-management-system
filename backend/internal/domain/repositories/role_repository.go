package repositories

import (
	"context"

	"github.com/sohamify/rms-backend/internal/domain/entities"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type RoleRepository interface {
	FindByID(ctx context.Context, id primitive.ObjectID) (*entities.Role, error)
}
