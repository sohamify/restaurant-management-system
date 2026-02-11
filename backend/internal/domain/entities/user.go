package entities

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	FirstName    string             `bson:"first_name" json:"first_name"`
	LastName     string             `bson:"last_name" json:"last_name"`
	Email        string             `bson:"email" json:"email"`
	PasswordHash string             `bson:"password_hash" json:"-"`
	RoleID       primitive.ObjectID `bson:"role_id" json:"role_id"`
	IsActive     bool               `bson:"is_active" json:"is_active"`
	Phone        string             `bson:"phone" json:"phone"`
	LastLoginAt  *time.Time         `bson:"last_login_at,omitempty" json:"last_login_at,omitempty"`
	CreatedAt    time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time          `bson:"updated_at" json:"updated_at"`
	DeletedAt    *time.Time         `bson:"deleted_at,omitempty" json:"-"`
}

// UserDTO for API responses (mapper)
type UserDTO struct {
	ID          string    `json:"id"`
	FirstName   string    `json:"first_name"`
	LastName    string    `json:"last_name"`
	Email       string    `json:"email"`
	RoleID      string    `json:"role_id"`
	IsActive    bool      `json:"is_active"`
	Phone       string    `json:"phone"`
	LastLoginAt time.Time `json:"last_login_at,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (u *User) ToDTO() *UserDTO {
	return &UserDTO{
		ID:          u.ID.Hex(),
		FirstName:   u.FirstName,
		LastName:    u.LastName,
		Email:       u.Email,
		RoleID:      u.RoleID.Hex(),
		IsActive:    u.IsActive,
		Phone:       u.Phone,
		LastLoginAt: *u.LastLoginAt,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
}
