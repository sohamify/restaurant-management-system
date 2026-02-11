package entities

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Role struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Name        string             `bson:"name" json:"name"`
	Permissions []string           `bson:"permissions" json:"permissions"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
}

// RoleDTO (if needed, but for now, role is simple)
type RoleDTO struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Permissions []string  `json:"permissions"`
	CreatedAt   time.Time `json:"created_at"`
}

func (r *Role) ToDTO() *RoleDTO {
	return &RoleDTO{
		ID:          r.ID.Hex(),
		Name:        r.Name,
		Permissions: r.Permissions,
		CreatedAt:   r.CreatedAt,
	}
}
