package services

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"github.com/sohamify/rms-backend/internal/domain/entities"
	"github.com/sohamify/rms-backend/internal/domain/repositories"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserInactive       = errors.New("user is inactive")
)

type AuthService interface {
	Login(ctx context.Context, email, password string) (*entities.UserDTO, string, error)
}

type authService struct {
	userRepo repositories.UserRepository
	roleRepo repositories.RoleRepository
	logger   *zap.Logger
}

func NewAuthService(userRepo repositories.UserRepository, roleRepo repositories.RoleRepository, logger *zap.Logger) AuthService {
	return &authService{userRepo, roleRepo, logger}
}

func (s *authService) Login(ctx context.Context, email, password string) (*entities.UserDTO, string, error) {
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		s.logger.Error("Failed to find user", zap.String("email", email), zap.Error(err))
		return nil, "", ErrInvalidCredentials
	}
	if user == nil {
		return nil, "", ErrInvalidCredentials
	}

	if !user.IsActive {
		return nil, "", ErrUserInactive
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		s.logger.Warn("Invalid password attempt", zap.String("email", email))
		return nil, "", ErrInvalidCredentials
	}

	// Fetch role for claims (including permissions)
	role, err := s.roleRepo.FindByID(ctx, user.RoleID)
	if err != nil || role == nil {
		s.logger.Error("Failed to find role", zap.String("role_id", user.RoleID.Hex()), zap.Error(err))
		return nil, "", errors.New("role not found")
	}

	// Update last_login_at using the generic Update method
	now := time.Now()
	updateFields := bson.M{
		"last_login_at": now,
	}

	if err := s.userRepo.Update(ctx, user.ID, updateFields); err != nil {
		s.logger.Error("Failed to update last login",
			zap.String("user_id", user.ID.Hex()),
			zap.Error(err),
		)
		// This is non-fatal — we continue with login anyway
	}

	// Generate JWT with permissions in claims (for RBAC middleware)
	claims := jwt.MapClaims{
		"user_id":     user.ID.Hex(),
		"role":        role.Name,
		"permissions": role.Permissions,
		"exp":         time.Now().Add(24 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		s.logger.Error("Failed to sign JWT", zap.Error(err))
		return nil, "", errors.New("internal error")
	}

	s.logger.Info("User logged in successfully",
		zap.String("user_id", user.ID.Hex()),
		zap.String("email", email),
		zap.String("role", role.Name),
	)

	return user.ToDTO(), signedToken, nil
}
