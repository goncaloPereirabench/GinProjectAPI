package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"ginprojectapi/internal/domain"
	"ginprojectapi/internal/store"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidCredentials = errors.New("invalid email or password")

type AuthService struct {
	users store.UserRepository
	jwt   *JWTManager
}

type AuthToken struct {
	AccessToken string
	TokenType   string
	ExpiresAt   time.Time
}

func NewAuthService(users store.UserRepository, jwt *JWTManager) *AuthService {
	return &AuthService{users: users, jwt: jwt}
}

func (s *AuthService) Register(ctx context.Context, email, password string) (domain.User, AuthToken, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return domain.User{}, AuthToken{}, err
	}

	now := time.Now().UTC()
	user := domain.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: string(hash),
		Role:         "customer",
		CreatedAt:    now,
	}

	if err := s.users.Create(ctx, user); err != nil {
		return domain.User{}, AuthToken{}, err
	}

	token, err := s.issueToken(user)
	return user, token, err
}

func (s *AuthService) Login(ctx context.Context, email, password string) (domain.User, AuthToken, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return domain.User{}, AuthToken{}, ErrInvalidCredentials
		}
		return domain.User{}, AuthToken{}, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return domain.User{}, AuthToken{}, ErrInvalidCredentials
	}

	token, err := s.issueToken(user)
	return user, token, err
}

func (s *AuthService) issueToken(user domain.User) (AuthToken, error) {
	accessToken, expiresAt, err := s.jwt.Generate(user.ID, user.Email, user.Role)
	if err != nil {
		return AuthToken{}, err
	}
	return AuthToken{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresAt:   expiresAt,
	}, nil
}
