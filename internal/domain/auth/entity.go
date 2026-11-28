package auth

import "time"

const (
	VerificationCodeLength   = 6
	VerificationCodeTTL      = 5 * time.Minute
	VerificationResendWindow = 2 * time.Minute
	MaxVerificationAttempts  = 5
)

// RefreshToken представляет сущность долгоживущего токена обновления сессии.
type RefreshToken struct {
	ID        int64
	UserID    int64
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}

// NewRefreshToken создаёт новый refresh token.
func NewRefreshToken(userID int64, tokenHash string, expiresAt time.Time) *RefreshToken {
	return &RefreshToken{
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now().UTC(),
	}
}

// IsRevoked возвращает true, если токен был отозван.
func (rt *RefreshToken) IsRevoked() bool {
	return rt.RevokedAt != nil
}

// IsExpired возвращает true, если срок действия токена истёк.
func (rt *RefreshToken) IsExpired(now time.Time) bool {
	return !now.Before(rt.ExpiresAt)
}

// EmailVerificationCode представляет одноразовый код подтверждения email при регистрации.
type EmailVerificationCode struct {
	ID          int64
	Email       string
	CodeHash    string
	ExpiresAt   time.Time
	LastSentAt  time.Time
	Attempts    int
	MaxAttempts int
	UsedAt      *time.Time
	CreatedAt   time.Time
}

// IsUsed возвращает true, если код уже был применён.
func (c *EmailVerificationCode) IsUsed() bool {
	return c.UsedAt != nil
}

// IsExpired возвращает true, если срок действия кода истёк.
func (c *EmailVerificationCode) IsExpired(now time.Time) bool {
	return !now.Before(c.ExpiresAt)
}

// IsExhausted возвращает true, если исчерпан лимит попыток проверки.
func (c *EmailVerificationCode) IsExhausted() bool {
	limit := c.MaxAttempts
	if limit <= 0 {
		limit = MaxVerificationAttempts
	}
	return c.Attempts >= limit
}

// CanResend проверяет окно повторной отправки и возвращает оставшееся число секунд до разблокировки.
func (c *EmailVerificationCode) CanResend(now time.Time) (bool, int) {
	nextAllowed := c.LastSentAt.Add(VerificationResendWindow)
	if !now.Before(nextAllowed) {
		return true, 0
	}
	remaining := nextAllowed.Sub(now)
	secs := int((remaining + time.Second - 1) / time.Second)
	if secs < 1 {
		secs = 1
	}
	return false, secs
}
