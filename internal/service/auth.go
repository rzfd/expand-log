package service

import (
	"context"
	"net/http"
	"net/mail"
	"regexp"
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
	email = normalizeEmail(ctx, email)
	if err := validateEmailForAuth(ctx, email); err != nil {
		logger.Warn().Err(err).Msg("service auth register email validation failed")
		return nil, err
	}
	if err := validatePasswordPolicy(ctx, password); err != nil {
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
	email = normalizeEmail(ctx, email)
	if err := validateEmailForAuth(ctx, email); err != nil {
		logger.Warn().Err(err).Msg("service auth login email validation failed")
		return nil, err
	}
	if strings.TrimSpace(password) == "" {
		logger.Warn().Msg("service auth login empty password")
		return nil, apperror.New(http.StatusBadRequest, "validation_error", "password is required")
	}

	if len(password) < 8 {
		logger.Warn().Msg("service auth login short password")
		return nil, apperror.New(http.StatusBadRequest, "validation_error", "password must be at least 8 characters")
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

func validateEmailForAuth(ctx context.Context, email string) error {
	logger := logging.FromContext(ctx)
	logger.Info().Msg("service auth validate email started")
	if email == "" {
		logger.Warn().Msg("service auth validate email empty")
		return apperror.New(http.StatusBadRequest, "validation_error", "email is required")
	}
	if strings.Count(email, "@") != 1 {
		logger.Warn().Str("email", email).Msg("service auth validate email invalid @ count")
		return apperror.New(http.StatusBadRequest, "validation_error", "email must be a valid address")
	}
	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed == nil || parsed.Address != email {
		logger.Warn().Err(err).Str("email", email).Msg("service auth validate email parse failed")
		return apperror.New(http.StatusBadRequest, "validation_error", "email must be a valid address")
	}
	logger.Info().Msg("service auth validate email completed")
	return nil
}

func normalizeEmail(ctx context.Context, email string) string {
	normalized := strings.ToLower(strings.TrimSpace(email))
	logging.FromContext(ctx).Info().Msg("service auth normalize email completed")
	return normalized
}

var (
	passwordHasLetter = regexp.MustCompile(`[A-Za-z]`)
	passwordHasNumber = regexp.MustCompile(`[0-9]`)
)

func validatePasswordPolicy(ctx context.Context, password string) error {
	logger := logging.FromContext(ctx)
	logger.Info().Msg("service auth validate password policy started")
	if len(password) < 8 {
		logger.Warn().Msg("service auth validate password policy too short")
		return apperror.New(http.StatusBadRequest, "validation_error", "password must be at least 8 characters")
	}
	if !passwordHasLetter.MatchString(password) || !passwordHasNumber.MatchString(password) {
		logger.Warn().Msg("service auth validate password policy missing character class")
		return apperror.New(http.StatusBadRequest, "validation_error", "password must include letters and numbers")
	}
	logger.Info().Msg("service auth validate password policy completed")
	return nil
}
