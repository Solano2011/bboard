package auth

import (
	"context"
	"fmt"
	"time"

	domainAuth "bboard/internal/domain/auth"
	"bboard/internal/domain/user"
	"bboard/pkg/token"
)

// TokenGenerator описывает контракт для выпуска access-токенов (JWT).
type TokenGenerator interface {
	GenerateToken(userID int64) (string, error)
}

// AuthOutput — результат успешной аутентификации или обновления токенов.
type AuthOutput struct {
	AccessToken  string
	RefreshToken string
	User         *user.User
}

func issueTokenPair(
	ctx context.Context,
	userID int64,
	tokenGen TokenGenerator,
	refreshRepo domainAuth.RefreshTokenRepository,
	refreshTTL time.Duration,
) (string, string, error) {
	accessToken, err := tokenGen.GenerateToken(userID)
	if err != nil {
		return "", "", fmt.Errorf("generate access token: %w", err)
	}

	rawRefresh, refreshHash, err := token.GenerateRefreshToken()
	if err != nil {
		return "", "", fmt.Errorf("generate refresh token: %w", err)
	}

	expiresAt := time.Now().UTC().Add(refreshTTL)
	rt := domainAuth.NewRefreshToken(userID, refreshHash, expiresAt)
	if err := refreshRepo.Create(ctx, rt); err != nil {
		return "", "", fmt.Errorf("save refresh token: %w", err)
	}

	return accessToken, rawRefresh, nil
}
