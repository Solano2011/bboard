package auth

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrRefreshTokenExpired = errors.New("refresh token has expired")
	ErrRefreshTokenRevoked = errors.New("refresh token has been revoked")

	ErrPasswordMismatch          = errors.New("passwords do not match")
	ErrInvalidVerificationCode   = errors.New("invalid verification code")
	ErrVerificationCodeNotFound  = errors.New("verification code not found")
	ErrVerificationCodeExpired   = errors.New("verification code has expired")
	ErrVerificationCodeExhausted = errors.New("verification code attempts exhausted")
	ErrVerificationResendTooSoon = errors.New("verification code requested too soon")
	ErrEmailDeliveryFailed       = errors.New("failed to send verification email")
)

// ResendCooldownError содержит оставшееся время до повторной отправки кода подтверждения.
type ResendCooldownError struct {
	RetryAfterSeconds int
}

func (e *ResendCooldownError) Error() string {
	return fmt.Sprintf("verification code requested too soon: retry after %d seconds", e.RetryAfterSeconds)
}

func (e *ResendCooldownError) Unwrap() error {
	return ErrVerificationResendTooSoon
}
