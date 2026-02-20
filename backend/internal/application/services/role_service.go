package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"

	"github.com/sohamify/rms-backend/internal/domain/entities"
	"github.com/sohamify/rms-backend/internal/domain/repositories"
)

var (
	ErrRoleAlreadyExists = errors.New("role name already exists")
	ErrRoleNotFound      = errors.New("role not found")
	ErrCannotDeleteAdmin = errors.New("cannot delete built-in ADMIN role")
	ErrRoleInUseByUsers  = errors.New("cannot delete role in use by users")
)

type RoleService interface {
	CreateRole(ctx context.Context, req *entities.CreateRoleRequest) (*entities.RoleDTO, error)
	ListRoles(ctx context.Context, page, limit int, search string) ([]*entities.RoleDTO, int64, error)
	GetRole(ctx context.Context, id string) (*entities.RoleDTO, error)
	UpdateRole(ctx context.Context, id string, req *entities.UpdateRoleRequest) (*entities.RoleDTO, error)
	DeleteRole(ctx context.Context, id string) error
}

type roleService struct {
	roleRepo repositories.RoleRepository
	userRepo repositories.UserRepository // injected for delete checks
	logger   *zap.Logger
}

func NewRoleService(roleRepo repositories.RoleRepository, userRepo repositories.UserRepository, logger *zap.Logger) RoleService {
	return &roleService{roleRepo, userRepo, logger}
}

func (s *roleService) CreateRole(ctx context.Context, req *entities.CreateRoleRequest) (*entities.RoleDTO, error) {
	// Check uniqueness
	existing, err := s.roleRepo.FindByName(ctx, req.Name)
	if err != nil {
		s.logger.Error("Failed to check role name uniqueness", zap.Error(err))
		return nil, errors.New("internal error")
	}
	if existing != nil {
		return nil, ErrRoleAlreadyExists
	}

	if len(req.Permissions) == 0 {
		return nil, errors.New("permissions cannot be empty")
	}

	role := &entities.Role{
		Name:        strings.ToUpper(req.Name),
		Permissions: req.Permissions,
		CreatedAt:   time.Now(),
	}

	// Create and get back the role with real ID
	createErr := s.roleRepo.Create(ctx, role)
	if createErr != nil {
		s.logger.Error("Failed to create role", zap.Error(createErr))
		return nil, createErr
	}
	createdRole := role

	s.logger.Info("Role created",
		zap.String("name", createdRole.Name),
		zap.String("id", createdRole.ID.Hex()),
	)

	return createdRole.ToDTO(), nil
}

func (s *roleService) ListRoles(ctx context.Context, page, limit int, search string) ([]*entities.RoleDTO, int64, error) {
	filter := bson.M{}

	if search != "" {
		filter["name"] = bson.M{"$regex": search, "$options": "i"}
	}

	opts := options.Find().
		SetSkip(int64((page - 1) * limit)).
		SetLimit(int64(limit)).
		SetSort(bson.M{"created_at": -1})

	roles, total, err := s.roleRepo.ListAll(ctx, filter, opts)
	if err != nil {
		s.logger.Error("Failed to list roles", zap.Error(err))
		return nil, 0, err
	}

	dtos := make([]*entities.RoleDTO, len(roles))
	for i, r := range roles {
		dtos[i] = r.ToDTO()
	}

	return dtos, total, nil
}

func (s *roleService) GetRole(ctx context.Context, id string) (*entities.RoleDTO, error) {
	rid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrRoleNotFound
	}

	role, err := s.roleRepo.FindByID(ctx, rid)
	if err != nil {
		s.logger.Error("Failed to get role", zap.String("id", id), zap.Error(err))
		return nil, err
	}
	if role == nil {
		return nil, ErrRoleNotFound
	}

	return role.ToDTO(), nil
}

func (s *roleService) UpdateRole(ctx context.Context, id string, req *entities.UpdateRoleRequest) (*entities.RoleDTO, error) {
	rid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrRoleNotFound
	}

	role, err := s.roleRepo.FindByID(ctx, rid)
	if err != nil || role == nil {
		return nil, ErrRoleNotFound
	}

	updateFields := bson.M{}

	if req.Name != nil {
		// Check uniqueness if changing name
		existing, _ := s.roleRepo.FindByName(ctx, *req.Name)
		if existing != nil && existing.ID != rid {
			return nil, ErrRoleAlreadyExists
		}
		updateFields["name"] = strings.ToUpper(*req.Name)
	}

	if req.Permissions != nil { // Check for nil instead of len > 0
		updateFields["permissions"] = req.Permissions
	}

	if len(updateFields) == 0 {
		return role.ToDTO(), nil // no changes
	}

	err = s.roleRepo.Update(ctx, rid, updateFields)
	if err != nil {
		s.logger.Error("Failed to update role", zap.String("id", id), zap.Error(err))
		return nil, err
	}

	updatedRole, err := s.roleRepo.FindByID(ctx, rid)
	if err != nil || updatedRole == nil {
		return nil, ErrRoleNotFound
	}

	s.logger.Info("Role updated", zap.String("id", id))
	return updatedRole.ToDTO(), nil
}

func (s *roleService) DeleteRole(ctx context.Context, id string) error {
	rid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return ErrRoleNotFound
	}

	role, err := s.roleRepo.FindByID(ctx, rid)
	if err != nil || role == nil {
		return ErrRoleNotFound
	}

	if role.Name == "ADMIN" {
		return ErrCannotDeleteAdmin
	}

	// Check if any users use this role
	usersFilter := bson.M{"role_id": rid, "deleted_at": nil}
	_, count, err := s.userRepo.FindAll(ctx, usersFilter, nil)
	if err != nil {
		s.logger.Error("Failed to count users for role", zap.String("role_id", id), zap.Error(err))
		return err
	}
	if count > 0 {
		return ErrRoleInUseByUsers
	}

	if err := s.roleRepo.Delete(ctx, rid); err != nil {
		s.logger.Error("Failed to delete role", zap.String("id", id), zap.Error(err))
		return err
	}

	s.logger.Info("Role deleted", zap.String("id", id))
	return nil
}
