package service

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/rzfd/expand/internal/model"
	"github.com/rzfd/expand/internal/pkg/apperror"
	"github.com/rzfd/expand/internal/pkg/auth"
)

type fakeAuthUserRepo struct {
	user      *model.User
	createErr error
}

func (f *fakeAuthUserRepo) Create(_ context.Context, user *model.User) error {
	if f.createErr != nil {
		return f.createErr
	}
	user.ID = 1
	user.CreatedAt = time.Now().UTC()
	user.UpdatedAt = user.CreatedAt
	f.user = user
	return nil
}

func (f *fakeAuthUserRepo) GetByEmail(_ context.Context, _ string) (*model.User, error) {
	return f.user, nil
}

func TestAuthServiceRegister(t *testing.T) {
	repo := &fakeAuthUserRepo{}
	tokenManager := auth.NewTokenManager("test-secret", time.Hour)
	service := NewAuthService(repo, tokenManager)

	user, err := service.Register(context.Background(), "test@example.com", "SecurePass123")
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	if user.ID == 0 {
		t.Fatalf("expected user ID to be set")
	}
	if user.Email != "test@example.com" {
		t.Fatalf("expected user email to be set")
	}
}

func TestAuthServiceLoginInvalidPassword(t *testing.T) {
	hashed, err := auth.HashPassword("password123")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}

	repo := &fakeAuthUserRepo{
		user: &model.User{
			ID:           1,
			Email:        "test@example.com",
			PasswordHash: hashed,
		},
	}

	tokenManager := auth.NewTokenManager("test-secret", time.Hour)
	service := NewAuthService(repo, tokenManager)

	if _, err := service.Login(context.Background(), "test@example.com", "wrong-password"); err == nil {
		t.Fatalf("expected login error for invalid password")
	}
}

func TestAuthServiceRegisterValidation(t *testing.T) {
	repo := &fakeAuthUserRepo{}
	tokenManager := auth.NewTokenManager("test-secret", time.Hour)
	service := NewAuthService(repo, tokenManager)

	tests := []struct {
		name    string
		email   string
		pass    string
		message string
	}{
		{
			name:    "invalid email format",
			email:   "invalid-email",
			pass:    "password123",
			message: "email must be a valid address",
		},
		{
			name:    "email too long",
			email:   "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa@example.com",
			pass:    "Password123",
			message: "email must be at most 254 characters",
		},
		{
			name:    "password missing number",
			email:   "test@example.com",
			pass:    "password",
			message: "password must include letters and numbers",
		},
		{
			name:    "password too short",
			email:   "test@example.com",
			pass:    "abc123",
			message: "password must be at least 8 characters",
		},
		{
			name:    "password too long",
			email:   "test@example.com",
			pass:    "Password123Password123Password123Password123Password123Password123Password123",
			message: "password must be at most 72 characters",
		},
		{
			name:    "common password blocked",
			email:   "test@example.com",
			pass:    "password123",
			message: "password is too common",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.Register(context.Background(), tt.email, tt.pass)
			if err == nil {
				t.Fatalf("expected validation error")
			}

			var appErr *apperror.Error
			if !errors.As(err, &appErr) {
				t.Fatalf("expected app error, got %T", err)
			}
			if appErr.Status != http.StatusBadRequest {
				t.Fatalf("expected status 400, got %d", appErr.Status)
			}
			if appErr.Code != "validation_error" {
				t.Fatalf("expected code validation_error, got %s", appErr.Code)
			}
			if appErr.Message != tt.message {
				t.Fatalf("expected message %q, got %q", tt.message, appErr.Message)
			}
		})
	}
}
