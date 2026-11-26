package auth

import "time"

// RefreshToken представляет сохранённый хэш refresh-токена пользователя.
type RefreshToken struct {
	ID        int64
	UserID    int64
	TokenHash string
	ExpiresAt time.Time
	CreatedAt time.Time
	RevokedAt *time.Time
}

// NewRefreshToken создаёт новую запись refresh-токена.
func NewRefreshToken(userID int64, tokenHash string, expiresAt time.Time) *RefreshToken {
	return &RefreshToken{
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now().UTC(),
	}
}

// IsRevoked проверяет, был ли токен отозван ранее.
func (rt *RefreshToken) IsRevoked() bool {
	return rt.RevokedAt != nil
}

// IsExpired проверяет, истёк ли срок действия токена на момент now.
func (rt *RefreshToken) IsExpired(now time.Time) bool {
	return !now.Before(rt.ExpiresAt)
}
