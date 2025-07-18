package database

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Connect(dbURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		return nil, err
	}
	return pool, nil
}
