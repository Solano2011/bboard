package auth

import (
	"context"
	"time"

	"bboard/internal/domain/user"
)

// RefreshTokenRepository определяет контракт хранения и отзыва refresh-токенов.
type RefreshTokenRepository interface {
	Create(ctx context.Context, rt *RefreshToken) error
	GetByHash(ctx context.Context, tokenHash string) (*RefreshToken, error)
	RevokeIfActive(ctx context.Context, id int64, revokedAt time.Time) (bool, error)
	RevokeByHash(ctx context.Context, tokenHash string, revokedAt time.Time) error
	RevokeAllByUserID(ctx context.Context, userID int64, revokedAt time.Time) error
}

// EmailVerificationRepository определяет контракт атомарной отправки и проверки кодов подтверждения email.
type EmailVerificationRepository interface {
	SaveAndSendCode(ctx context.Context, email, codeHash string, now, expiresAt time.Time, maxAttempts int, sendFn func() error) error
	VerifyCodeAndCreateUser(ctx context.Context, email, codeHash string, now time.Time, u *user.User) error
}

// EmailSender отправляет письма с кодами подтверждения.
type EmailSender interface {
	SendVerificationCode(ctx context.Context, toEmail, code string) error
}
