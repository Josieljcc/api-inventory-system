package users

import (
	"context"
	"errors"

	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	Repo RepositoryInterface
}

func NewService(repo RepositoryInterface) *Service {
	return &Service{Repo: repo}
}

func (s *Service) Register(ctx context.Context, username, password, role string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.Repo.CreateUser(ctx, username, string(hash), role)
}

func (s *Service) Authenticate(ctx context.Context, username, password string) (*User, error) {
	u, err := s.Repo.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		return nil, errors.New("invalid password")
	}
	return u, nil
}

func (s *Service) GetByUsername(ctx context.Context, username string) (*User, error) {
	return s.Repo.GetUserByUsername(ctx, username)
}

func (s *Service) GenerateRefreshToken(ctx context.Context, userID int) (string, error) {
	token := uuid.NewString()
	expiresAt := time.Now().Add(7 * 24 * time.Hour) // 7 dias
	if err := s.Repo.CreateRefreshToken(ctx, userID, token, expiresAt); err != nil {
		return "", err
	}
	return token, nil
}

func (s *Service) ValidateRefreshToken(ctx context.Context, token string) (*RefreshToken, error) {
	rt, err := s.Repo.GetRefreshToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if rt.ExpiresAt.Before(time.Now()) {
		_ = s.Repo.DeleteRefreshToken(ctx, token)
		return nil, errors.New("refresh token expired")
	}
	return rt, nil
}

func (s *Service) RotateRefreshToken(ctx context.Context, oldToken string) (string, error) {
	rt, err := s.ValidateRefreshToken(ctx, oldToken)
	if err != nil {
		return "", err
	}
	_ = s.Repo.DeleteRefreshToken(ctx, oldToken)
	return s.GenerateRefreshToken(ctx, rt.UserID)
}
