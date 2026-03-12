package service

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/rzfd/expand/internal/model"
	"github.com/rzfd/expand/internal/pkg/apperror"
	"github.com/rzfd/expand/internal/pkg/auth"
	"github.com/rzfd/expand/internal/pkg/logging"
	"github.com/rzfd/expand/internal/repository"
)

type authUserRepository interface {
	Create(ctx context.Context, user *model.User) error
	GetByEmail(ctx context.Context, email string) (*model.User, error)
}

type AuthService struct {
	users        authUserRepository
	tokenManager *auth.TokenManager
}

type AuthResult struct {
	User      model.User
	Token     string
	ExpiresAt time.Time
}

func NewAuthService(users authUserRepository, tokenManager *auth.TokenManager) *AuthService {
	return &AuthService{
		users:        users,
		tokenManager: tokenManager,
	}
}

func (s *AuthService) Register(ctx context.Context, email, password string) (*model.User, error) {
	logger := logging.FromContext(ctx)
	logger.Info().Msg("service auth register started")
	email = normalizeEmail(email)
	if err := validateEmailAndPassword(email, password); err != nil {
		logger.Warn().Err(err).Msg("service auth register validation failed")
		return nil, err
	}

	existing, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		logger.Error().Err(err).Msg("service auth register user lookup failed")
		return nil, apperror.Wrap(http.StatusInternalServerError, "internal_error", "failed to check existing user", err)
	}
	if existing != nil {
		logger.Warn().Msg("service auth register email already exists")
		return nil, apperror.New(http.StatusConflict, "email_exists", "email is already registered")
	}

	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		logger.Error().Err(err).Msg("service auth register password hash failed")
		return nil, apperror.Wrap(http.StatusInternalServerError, "internal_error", "failed to hash password", err)
	}

	user := model.User{
		Email:        email,
		PasswordHash: passwordHash,
	}
	if err := s.users.Create(ctx, &user); err != nil {
		if repository.IsUniqueViolation(err) {
			logger.Warn().Err(err).Msg("service auth register unique violation")
			return nil, apperror.New(http.StatusConflict, "email_exists", "email is already registered")
		}
		logger.Error().Err(err).Msg("service auth register user create failed")
		return nil, apperror.Wrap(http.StatusInternalServerError, "internal_error", "failed to create user", err)
	}

	logger.Info().Int64("user_id", user.ID).Msg("service auth register completed")
	return &user, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*AuthResult, error) {
	logger := logging.FromContext(ctx)
	logger.Info().Msg("service auth login started")
	email = normalizeEmail(email)
	if err := validateEmailAndPassword(email, password); err != nil {
		logger.Warn().Err(err).Msg("service auth login validation failed")
		return nil, err
	}

	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		logger.Error().Err(err).Msg("service auth login user lookup failed")
		return nil, apperror.Wrap(http.StatusInternalServerError, "internal_error", "failed to load user", err)
	}
	if user == nil {
		logger.Warn().Msg("service auth login user not found")
		return nil, apperror.New(http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
	}

	if err := auth.ComparePassword(user.PasswordHash, password); err != nil {
		logger.Warn().Err(err).Int64("user_id", user.ID).Msg("service auth login password mismatch")
		return nil, apperror.New(http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
	}

	token, expiresAt, err := s.tokenManager.Generate(user.ID)
	if err != nil {
		logger.Error().Err(err).Int64("user_id", user.ID).Msg("service auth login token generation failed")
		return nil, apperror.Wrap(http.StatusInternalServerError, "internal_error", "failed to generate token", err)
	}

	logger.Info().Int64("user_id", user.ID).Msg("service auth login completed")
	return &AuthResult{
		User:      *user,
		Token:     token,
		ExpiresAt: expiresAt,
	}, nil
}

func validateEmailAndPassword(email, password string) error {
	logger := logging.FromContext(nil)
	logger.Info().Msg("service auth validate credentials started")
	if email == "" || !strings.Contains(email, "@") {
		logger.Warn().Msg("service auth validate credentials invalid email")
		return apperror.New(http.StatusBadRequest, "validation_error", "email must be valid")
	}
	if len(password) < 8 {
		logger.Warn().Msg("service auth validate credentials short password")
		return apperror.New(http.StatusBadRequest, "validation_error", "password must be at least 8 characters")
	}
	logger.Info().Msg("service auth validate credentials completed")
	return nil
}

func normalizeEmail(email string) string {
	normalized := strings.ToLower(strings.TrimSpace(email))
	logging.FromContext(nil).Info().Msg("service auth normalize email completed")
	return normalized
}
