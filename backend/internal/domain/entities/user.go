package entities

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ── Domain Entity ────────────────────────────────────────────────────────

type User struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	FirstName    string             `bson:"first_name" json:"first_name"`
	LastName     string             `bson:"last_name" json:"last_name"`
	Email        string             `bson:"email" json:"email"`
	PasswordHash string             `bson:"password_hash" json:"-"` // never expose
	RoleID       primitive.ObjectID `bson:"role_id" json:"role_id"`
	IsActive     bool               `bson:"is_active" json:"is_active"`
	Phone        string             `bson:"phone" json:"phone"`
	LastLoginAt  *time.Time         `bson:"last_login_at,omitempty" json:"last_login_at,omitempty"`
	CreatedAt    time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time          `bson:"updated_at" json:"updated_at"`
	DeletedAt    *time.Time         `bson:"deleted_at,omitempty" json:"deleted_at,omitempty"`
}

// ── API Output Shape ─────────────────────────────────────────────────────

type UserDTO struct {
	ID          string     `json:"id"`
	FirstName   string     `json:"first_name"`
	LastName    string     `json:"last_name"`
	Email       string     `json:"email"`
	RoleID      string     `json:"role_id"`
	IsActive    bool       `json:"is_active"`
	Phone       string     `json:"phone"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (u *User) ToDTO() *UserDTO {
	dto := &UserDTO{
		ID:        u.ID.Hex(),
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Email:     u.Email,
		RoleID:    u.RoleID.Hex(),
		IsActive:  u.IsActive,
		Phone:     u.Phone,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
	if u.LastLoginAt != nil {
		dto.LastLoginAt = u.LastLoginAt
	}
	return dto
}

// ── Input DTOs (for request binding + validation) ────────────────────────
// These can live here or in a future dto package — for now here is fine

type CreateUserRequest struct {
	FirstName string `json:"first_name" binding:"required,min=2,max=50"`
	LastName  string `json:"last_name"  binding:"required,min=2,max=50"`
	Email     string `json:"email"      binding:"required,email"`
	Password  string `json:"password"   binding:"required,min=8"`
	Phone     string `json:"phone"      binding:"omitempty"`
	RoleID    string `json:"role_id"    binding:"required"` // hex string
}

type UpdateUserRequest struct {
	FirstName *string `json:"first_name,omitempty"`
	LastName  *string `json:"last_name,omitempty"`
	Phone     *string `json:"phone,omitempty"`
	RoleID    *string `json:"role_id,omitempty"`
	IsActive  *bool   `json:"is_active,omitempty"`
}

type ChangePasswordRequest struct {
	OldPassword *string `json:"old_password,omitempty"` // required for self-change
	NewPassword string  `json:"new_password" binding:"required,min=8"`
}
