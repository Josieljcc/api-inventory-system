package database

import (
	"context"
	"io/ioutil"

	"github.com/jackc/pgx/v5/pgxpool"
)

func RunMigrations(db *pgxpool.Pool) error {
	migrationFile := "internal/database/migrations.sql"
	b, err := ioutil.ReadFile(migrationFile)
	if err != nil {
		return err
	}
	_, err = db.Exec(context.Background(), string(b))
	return err
}
