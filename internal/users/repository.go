package users

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	DB *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{DB: db}
}

func (r *Repository) CreateUser(ctx context.Context, username, passwordHash, role string) error {
	_, err := r.DB.Exec(ctx, `INSERT INTO users (username, password_hash, role) VALUES ($1, $2, $3)`, username, passwordHash, role)
	return err
}

func (r *Repository) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	var u User
	err := r.DB.QueryRow(ctx, `SELECT id, username, password_hash FROM users WHERE username=$1`, username).Scan(&u.ID, &u.Username, &u.PasswordHash)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *Repository) CreateRefreshToken(ctx context.Context, userID int, token string, expiresAt time.Time) error {
	_, err := r.DB.Exec(ctx, `INSERT INTO refresh_tokens (user_id, token, expires_at) VALUES ($1, $2, $3)`, userID, token, expiresAt)
	return err
}

func (r *Repository) GetRefreshToken(ctx context.Context, token string) (*RefreshToken, error) {
	var rt RefreshToken
	err := r.DB.QueryRow(ctx, `SELECT id, user_id, token, expires_at FROM refresh_tokens WHERE token=$1`, token).Scan(&rt.ID, &rt.UserID, &rt.Token, &rt.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return &rt, nil
}

func (r *Repository) DeleteRefreshToken(ctx context.Context, token string) error {
	_, err := r.DB.Exec(ctx, `DELETE FROM refresh_tokens WHERE token=$1`, token)
	return err
}

type RepositoryInterface interface {
	CreateUser(ctx context.Context, username, passwordHash, role string) error
	GetUserByUsername(ctx context.Context, username string) (*User, error)
	CreateRefreshToken(ctx context.Context, userID int, token string, expiresAt time.Time) error
	GetRefreshToken(ctx context.Context, token string) (*RefreshToken, error)
	DeleteRefreshToken(ctx context.Context, token string) error
}
