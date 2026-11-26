package auth

import (
	"context"
	"time"
)

// RefreshTokenRepository определяет контракт хранения и отзыва refresh-токенов.
type RefreshTokenRepository interface {
	Create(ctx context.Context, rt *RefreshToken) error
	GetByHash(ctx context.Context, tokenHash string) (*RefreshToken, error)
	// RevokeIfActive атомарно отзывает активный токен. Возвращает false, если токен уже был отозван.
	RevokeIfActive(ctx context.Context, id int64, revokedAt time.Time) (bool, error)
	RevokeByHash(ctx context.Context, tokenHash string, revokedAt time.Time) error
	RevokeAllByUserID(ctx context.Context, userID int64, revokedAt time.Time) error
}
