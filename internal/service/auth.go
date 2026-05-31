package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/realtime-chat/internal/domain"
	"github.com/realtime-chat/internal/pkg/hasher"
	"github.com/realtime-chat/internal/pkg/jwt"
)

type AuthService struct {
	users  domain.UserRepository
	hasher hasher.Hasher
	tokens *jwt.Manager
}

func NewAuthService(users domain.UserRepository, h hasher.Hasher, tokens *jwt.Manager) *AuthService {
	return &AuthService{users: users, hasher: h, tokens: tokens}
}

func (s *AuthService) Register(ctx context.Context, username, email, password string) (*domain.User, string, error) {
	username = strings.TrimSpace(username)
	email = strings.ToLower(strings.TrimSpace(email))

	if username == "" || email == "" || len(password) < 8 {
		return nil, "", domain.ErrInvalidInput
	}

	hash, err := s.hasher.Hash(password)
	if err != nil {
		return nil, "", fmt.Errorf("hash password: %w", err)
	}

	now := time.Now().UTC()
	u := &domain.User{
		ID:           uuid.New(),
		Username:     username,
		Email:        email,
		PasswordHash: hash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.users.Create(ctx, u); err != nil {
		return nil, "", err
	}

	token, err := s.tokens.Issue(u.ID)
	if err != nil {
		return nil, "", fmt.Errorf("issue token: %w", err)
	}
	return u, token, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*domain.User, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || password == "" {
		return nil, "", domain.ErrInvalidCredentials
	}

	u, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, "", domain.ErrInvalidCredentials
		}
		return nil, "", err
	}
	if err := s.hasher.Compare(u.PasswordHash, password); err != nil {
		return nil, "", domain.ErrInvalidCredentials
	}

	token, err := s.tokens.Issue(u.ID)
	if err != nil {
		return nil, "", fmt.Errorf("issue token: %w", err)
	}
	return u, token, nil
}

func (s *AuthService) ParseToken(token string) (uuid.UUID, error) {
	id, err := s.tokens.Parse(token)
	if err != nil {
		return uuid.Nil, domain.ErrUnauthorized
	}
	return id, nil
}
