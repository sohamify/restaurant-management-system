package repositories

import (
	"context"
	"time"

	"github.com/sohamify/rms-backend/internal/domain/entities"
)

type UserRepository interface {
	FindByEmail(ctx context.Context, email string) (*entities.User, error)
	UpdateLastLogin(ctx context.Context, id string, lastLogin time.Time) error
	// Add more like Create for registration later
}
