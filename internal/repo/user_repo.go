package repo

import (
	"context"
	"fmt"
	"time"

	"avagostar-form-server/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepo struct {
	pool    *pgxpool.Pool
	timeout time.Duration
}

func NewUserRepo(pool *pgxpool.Pool, timeout time.Duration) *UserRepo {
	return &UserRepo{pool: pool, timeout: timeout}
}

func (r *UserRepo) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	row := r.pool.QueryRow(ctx, `
		SELECT id, username, role, password_hash, reset_code_hash, reset_code_expires_at, created_at, updated_at
		FROM users
		WHERE username = $1
	`, username)

	var user models.User
	if err := row.Scan(
		&user.ID,
		&user.Username,
		&user.Role,
		&user.PasswordHash,
		&user.ResetCodeHash,
		&user.ResetCodeExpiresAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("get user by username: %w", err)
	}
	return &user, nil
}

func (r *UserRepo) GetByID(ctx context.Context, id string) (*models.User, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	row := r.pool.QueryRow(ctx, `
		SELECT id, username, role, password_hash, reset_code_hash, reset_code_expires_at, created_at, updated_at
		FROM users
		WHERE id = $1
	`, id)

	var user models.User
	if err := row.Scan(
		&user.ID,
		&user.Username,
		&user.Role,
		&user.PasswordHash,
		&user.ResetCodeHash,
		&user.ResetCodeExpiresAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return &user, nil
}

func (r *UserRepo) UpdateResetCode(ctx context.Context, userID string, codeHash *string, expiresAt *time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	_, err := r.pool.Exec(ctx, `
		UPDATE users
		SET reset_code_hash = $1, reset_code_expires_at = $2, updated_at = NOW()
		WHERE id = $3
	`, codeHash, expiresAt, userID)
	if err != nil {
		return fmt.Errorf("update reset code: %w", err)
	}
	return nil
}

func (r *UserRepo) UpdatePassword(ctx context.Context, userID, passwordHash string) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	_, err := r.pool.Exec(ctx, `
		UPDATE users
		SET password_hash = $1, reset_code_hash = NULL, reset_code_expires_at = NULL, updated_at = NOW()
		WHERE id = $2
	`, passwordHash, userID)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	return nil
}
