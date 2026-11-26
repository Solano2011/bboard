package auth

import "errors"

var (
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrRefreshTokenExpired = errors.New("refresh token has expired")
	ErrRefreshTokenRevoked = errors.New("refresh token has been revoked")
)
