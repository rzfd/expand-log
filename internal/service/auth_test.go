package service

import (
	"context"
	"testing"
	"time"

	"github.com/rzfd/expand/internal/model"
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

	user, err := service.Register(context.Background(), "test@example.com", "password123")
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
