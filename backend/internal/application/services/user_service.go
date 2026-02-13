// internal/application/services/user_service.go
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
	"golang.org/x/crypto/bcrypt"

	"github.com/sohamify/rms-backend/internal/domain/entities"
	"github.com/sohamify/rms-backend/internal/domain/repositories"
)

var (
	ErrEmailAlreadyExists    = errors.New("email already in use")
	ErrInvalidRole           = errors.New("invalid or inactive role")
	ErrUserNotFound          = errors.New("user not found")
	ErrCannotDeleteSelf      = errors.New("cannot delete your own account")
	ErrCannotDeleteLastAdmin = errors.New("cannot delete the last admin account")
	ErrOldPasswordRequired   = errors.New("old password is required for self-change")
	ErrInvalidOldPassword    = errors.New("old password is incorrect")
	ErrSameAsOldPassword     = errors.New("new password cannot be the same as old password")
)

type UserService interface {
	CreateUser(ctx context.Context, req *entities.CreateUserRequest, createdByID string) (*entities.UserDTO, error)
	ListUsers(ctx context.Context, page, limit int, roleFilter, search string, isActive *bool) ([]*entities.UserDTO, int64, error)
	GetUser(ctx context.Context, id string) (*entities.UserDTO, error)
	UpdateUser(ctx context.Context, id string, req *entities.UpdateUserRequest, updatedByID string) (*entities.UserDTO, error)
	DeleteUser(ctx context.Context, id string, deletedByID string) error
	ChangePassword(ctx context.Context, targetUserID string, req *entities.ChangePasswordRequest, requesterID string, isAdmin bool) error
}

type userService struct {
	userRepo repositories.UserRepository
	roleRepo repositories.RoleRepository
	logger   *zap.Logger
}

func NewUserService(userRepo repositories.UserRepository, roleRepo repositories.RoleRepository, logger *zap.Logger) UserService {
	return &userService{userRepo, roleRepo, logger}
}

// ── Create User ───────────────────────────────────────────────────────────────

func (s *userService) CreateUser(ctx context.Context, req *entities.CreateUserRequest, createdByID string) (*entities.UserDTO, error) {
	if err := ValidatePasswordStrength(req.Password); err != nil {
		return nil, err
	}

	// Check email uniqueness
	existing, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		s.logger.Error("Failed to check email uniqueness", zap.Error(err))
		return nil, errors.New("internal server error")
	}
	if existing != nil {
		return nil, ErrEmailAlreadyExists
	}

	// Validate role
	roleID, err := primitive.ObjectIDFromHex(req.RoleID)
	if err != nil {
		return nil, ErrInvalidRole
	}
	role, err := s.roleRepo.FindByID(ctx, roleID)
	if err != nil || role == nil {
		s.logger.Warn("Invalid role ID", zap.String("role_id", req.RoleID))
		return nil, ErrInvalidRole
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &entities.User{
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Email:        strings.ToLower(req.Email),
		PasswordHash: string(hash),
		RoleID:       roleID,
		IsActive:     true,
		Phone:        req.Phone,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		s.logger.Error("Failed to create user", zap.Error(err), zap.String("email", req.Email))
		return nil, err
	}

	// TODO: Audit log entry: "USER_CREATED" by createdByID

	s.logger.Info("User created successfully",
		zap.String("user_id", user.ID.Hex()),
		zap.String("email", req.Email),
		zap.String("created_by", createdByID),
	)

	return user.ToDTO(), nil
}

// ── List Users (with pagination, filters, search) ─────────────────────────────

func (s *userService) ListUsers(ctx context.Context, page, limit int, roleFilter, search string, isActive *bool) ([]*entities.UserDTO, int64, error) {
	filter := bson.M{"deleted_at": nil}

	if roleFilter != "" {
		rid, err := primitive.ObjectIDFromHex(roleFilter)
		if err == nil {
			filter["role_id"] = rid
		}
	}

	if isActive != nil {
		filter["is_active"] = *isActive
	}

	if search != "" {
		search = strings.TrimSpace(search)
		searchRegex := primitive.Regex{Pattern: search, Options: "i"}
		filter["$or"] = []bson.M{
			{"first_name": searchRegex},
			{"last_name": searchRegex},
			{"email": searchRegex},
		}
	}

	opts := options.Find().
		SetSkip(int64((page - 1) * limit)).
		SetLimit(int64(limit)).
		SetSort(bson.M{"created_at": -1})

	users, total, err := s.userRepo.FindAll(ctx, filter, opts)
	if err != nil {
		s.logger.Error("Failed to list users", zap.Error(err))
		return nil, 0, err
	}

	dtos := make([]*entities.UserDTO, len(users))
	for i, u := range users {
		dtos[i] = u.ToDTO()
	}

	return dtos, total, nil
}

// ── Get Single User ───────────────────────────────────────────────────────────

func (s *userService) GetUser(ctx context.Context, id string) (*entities.UserDTO, error) {
	uid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrUserNotFound
	}

	user, err := s.userRepo.FindByID(ctx, uid)
	if err != nil {
		s.logger.Error("Failed to get user", zap.String("id", id), zap.Error(err))
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	return user.ToDTO(), nil
}

// ── Update User ───────────────────────────────────────────────────────────────

func (s *userService) UpdateUser(ctx context.Context, id string, req *entities.UpdateUserRequest, updatedByID string) (*entities.UserDTO, error) {
	uid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrUserNotFound
	}

	user, err := s.userRepo.FindByID(ctx, uid)
	if err != nil || user == nil {
		return nil, ErrUserNotFound
	}

	update := bson.M{}

	if req.FirstName != nil {
		update["first_name"] = *req.FirstName
	}
	if req.LastName != nil {
		update["last_name"] = *req.LastName
	}
	if req.Phone != nil {
		update["phone"] = *req.Phone
	}
	if req.IsActive != nil {
		update["is_active"] = *req.IsActive
	}
	if req.RoleID != nil {
		rid, err := primitive.ObjectIDFromHex(*req.RoleID)
		if err != nil {
			return nil, ErrInvalidRole
		}
		role, _ := s.roleRepo.FindByID(ctx, rid)
		if role == nil {
			return nil, ErrInvalidRole
		}
		update["role_id"] = rid
	}

	if len(update) == 0 {
		return user.ToDTO(), nil // no changes
	}

	if err := s.userRepo.Update(ctx, uid, bson.M{"$set": update}); err != nil {
		s.logger.Error("Failed to update user", zap.String("id", id), zap.Error(err))
		return nil, err
	}

	// Refresh user
	updatedUser, _ := s.userRepo.FindByID(ctx, uid)
	// TODO: Audit log "USER_UPDATED"

	s.logger.Info("User updated", zap.String("id", id), zap.String("by", updatedByID))
	return updatedUser.ToDTO(), nil
}

// ── Delete User (soft) ────────────────────────────────────────────────────────

func (s *userService) DeleteUser(ctx context.Context, id string, deletedByID string) error {
	uid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return ErrUserNotFound
	}

	if deletedByID == id {
		return ErrCannotDeleteSelf
	}

	// Prevent deleting last admin
	user, err := s.userRepo.FindByID(ctx, uid)
	if err != nil || user == nil {
		return ErrUserNotFound
	}

	role, _ := s.roleRepo.FindByID(ctx, user.RoleID)
	if role != nil && role.Name == "ADMIN" {
		countFilter := bson.M{"role_id": user.RoleID, "deleted_at": nil, "is_active": true}
		_, count, _ := s.userRepo.FindAll(ctx, countFilter, nil)
		if count <= 1 {
			return ErrCannotDeleteLastAdmin
		}
	}

	if err := s.userRepo.SoftDelete(ctx, uid); err != nil {
		s.logger.Error("Failed to soft delete user", zap.String("id", id), zap.Error(err))
		return err
	}

	// TODO: Audit log "USER_DELETED"

	s.logger.Info("User soft-deleted", zap.String("id", id), zap.String("by", deletedByID))
	return nil
}

// ── Change Password ───────────────────────────────────────────────────────────

func (s *userService) ChangePassword(ctx context.Context, targetUserID string, req *entities.ChangePasswordRequest, requesterID string, isAdmin bool) error {
	tid, err := primitive.ObjectIDFromHex(targetUserID)
	if err != nil {
		return ErrUserNotFound
	}

	user, err := s.userRepo.FindByID(ctx, tid)
	if err != nil || user == nil {
		return ErrUserNotFound
	}

	// Self-change: must provide old password
	if !isAdmin {
		if req.OldPassword == nil || *req.OldPassword == "" {
			return ErrOldPasswordRequired
		}
		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(*req.OldPassword)); err != nil {
			return ErrInvalidOldPassword
		}
		if *req.OldPassword == req.NewPassword {
			return ErrSameAsOldPassword
		}
	}

	// Admin reset: no old password needed

	if err := ValidatePasswordStrength(req.NewPassword); err != nil {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	update := bson.M{
		"$set": bson.M{
			"password_hash": string(hash),
			"updated_at":    time.Now(),
		},
	}

	if err := s.userRepo.Update(ctx, tid, update); err != nil {
		s.logger.Error("Failed to change password", zap.String("user_id", targetUserID), zap.Error(err))
		return err
	}

	// TODO: Audit log "PASSWORD_CHANGED" or "PASSWORD_RESET"

	s.logger.Info("Password changed", zap.String("user_id", targetUserID), zap.Bool("by_admin", isAdmin))
	return nil
}
