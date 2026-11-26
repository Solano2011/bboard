package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"bboard/internal/domain/auth"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RefreshTokenRepo struct {
	pool *pgxpool.Pool
}

func NewRefreshTokenRepo(pool *pgxpool.Pool) *RefreshTokenRepo {
	return &RefreshTokenRepo{pool: pool}
}

func (r *RefreshTokenRepo) Create(ctx context.Context, rt *auth.RefreshToken) error {
	query := `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at, created_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`
	if err := r.pool.QueryRow(ctx, query, rt.UserID, rt.TokenHash, rt.ExpiresAt, rt.CreatedAt).Scan(&rt.ID); err != nil {
		return fmt.Errorf("insert refresh token: %w", err)
	}
	return nil
}

func (r *RefreshTokenRepo) GetByHash(ctx context.Context, tokenHash string) (*auth.RefreshToken, error) {
	query := `
		SELECT id, user_id, token_hash, expires_at, created_at, revoked_at
		FROM refresh_tokens
		WHERE token_hash = $1
	`
	var rt auth.RefreshToken
	err := r.pool.QueryRow(ctx, query, tokenHash).Scan(
		&rt.ID, &rt.UserID, &rt.TokenHash, &rt.ExpiresAt, &rt.CreatedAt, &rt.RevokedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, auth.ErrInvalidRefreshToken
		}
		return nil, fmt.Errorf("get refresh token by hash: %w", err)
	}
	return &rt, nil
}

func (r *RefreshTokenRepo) RevokeIfActive(ctx context.Context, id int64, revokedAt time.Time) (bool, error) {
	query := `
		UPDATE refresh_tokens
		SET revoked_at = $1
		WHERE id = $2 AND revoked_at IS NULL
	`
	cmdTag, err := r.pool.Exec(ctx, query, revokedAt, id)
	if err != nil {
		return false, fmt.Errorf("revoke refresh token by id: %w", err)
	}
	return cmdTag.RowsAffected() == 1, nil
}

func (r *RefreshTokenRepo) RevokeByHash(ctx context.Context, tokenHash string, revokedAt time.Time) error {
	query := `
		UPDATE refresh_tokens
		SET revoked_at = $1
		WHERE token_hash = $2 AND revoked_at IS NULL
	`
	if _, err := r.pool.Exec(ctx, query, revokedAt, tokenHash); err != nil {
		return fmt.Errorf("revoke refresh token by hash: %w", err)
	}
	return nil
}

func (r *RefreshTokenRepo) RevokeAllByUserID(ctx context.Context, userID int64, revokedAt time.Time) error {
	query := `
		UPDATE refresh_tokens
		SET revoked_at = $1
		WHERE user_id = $2 AND revoked_at IS NULL
	`
	if _, err := r.pool.Exec(ctx, query, revokedAt, userID); err != nil {
		return fmt.Errorf("revoke all user refresh tokens: %w", err)
	}
	return nil
}
