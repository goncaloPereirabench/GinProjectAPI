package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"ginprojectapi/internal/store"
	"ginprojectapi/internal/store/memory"
)

func TestAuthServiceRegisterAndLogin(t *testing.T) {
	repositories := memory.NewRepositories()
	jwtManager := NewJWTManager("test-secret", "test-suite", 15*time.Minute)
	auth := NewAuthService(repositories.Users, jwtManager)

	user, token, err := auth.Register(context.Background(), "Buyer@Example.com", "very-secret-password")
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if user.Email != "buyer@example.com" {
		t.Fatalf("email was not normalized: %s", user.Email)
	}
	if token.AccessToken == "" {
		t.Fatal("expected access token")
	}

	loggedIn, loginToken, err := auth.Login(context.Background(), "buyer@example.com", "very-secret-password")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if loggedIn.ID != user.ID {
		t.Fatalf("expected user %s, got %s", user.ID, loggedIn.ID)
	}
	if loginToken.AccessToken == "" {
		t.Fatal("expected login token")
	}
}

func TestAuthServiceRejectsInvalidPassword(t *testing.T) {
	repositories := memory.NewRepositories()
	auth := NewAuthService(repositories.Users, NewJWTManager("test-secret", "test-suite", 15*time.Minute))

	_, _, err := auth.Register(context.Background(), "buyer@example.com", "very-secret-password")
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	_, _, err = auth.Login(context.Background(), "buyer@example.com", "wrong-password")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected invalid credentials, got %v", err)
	}
}

func TestAuthServiceRejectsDuplicateEmail(t *testing.T) {
	repositories := memory.NewRepositories()
	auth := NewAuthService(repositories.Users, NewJWTManager("test-secret", "test-suite", 15*time.Minute))

	_, _, err := auth.Register(context.Background(), "buyer@example.com", "very-secret-password")
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	_, _, err = auth.Register(context.Background(), "buyer@example.com", "very-secret-password")
	if !errors.Is(err, store.ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}
