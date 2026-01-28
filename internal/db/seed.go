package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type SeedUser struct {
	Username string
	Password string
	Role     string
}

func EnsureSeedUsers(ctx context.Context, pool *pgxpool.Pool, timeout time.Duration) error {
	seeds := []SeedUser{
		{Username: "admin", Password: "admin123", Role: "admin"},
		{Username: "user1", Password: "1111", Role: "user"},
		{Username: "user2", Password: "2222", Role: "user"},
	}

	for _, seed := range seeds {
		exists, err := userExists(ctx, pool, timeout, seed.Username)
		if err != nil {
			return err
		}
		if exists {
			continue
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(seed.Password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("hash seed password: %w", err)
		}

		ctxInsert, cancel := context.WithTimeout(ctx, timeout)
		_, err = pool.Exec(ctxInsert, `
			INSERT INTO users (username, password_hash, role)
			VALUES ($1, $2, $3)
		`, seed.Username, string(hash), seed.Role)
		cancel()
		if err != nil {
			return fmt.Errorf("insert seed user %s: %w", seed.Username, err)
		}
	}

	return nil
}

func userExists(ctx context.Context, pool *pgxpool.Pool, timeout time.Duration, username string) (bool, error) {
	ctxCheck, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	row := pool.QueryRow(ctxCheck, "SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)", username)
	var exists bool
	if err := row.Scan(&exists); err != nil {
		return false, fmt.Errorf("check user exists: %w", err)
	}
	return exists, nil
}
