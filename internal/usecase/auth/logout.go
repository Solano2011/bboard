package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	domainAuth "bboard/internal/domain/auth"
	"bboard/pkg/token"

	"github.com/sirupsen/logrus"
)

type LogoutUseCase struct {
	refreshRepo domainAuth.RefreshTokenRepository
	log         *logrus.Entry
}

func NewLogoutUseCase(refreshRepo domainAuth.RefreshTokenRepository, log *logrus.Logger) *LogoutUseCase {
	return &LogoutUseCase{
		refreshRepo: refreshRepo,
		log:         log.WithField("usecase", "auth.logout"),
	}
}

func (uc *LogoutUseCase) Execute(ctx context.Context, rawRefreshToken string) error {
	if strings.TrimSpace(rawRefreshToken) == "" {
		return nil
	}

	tokenHash := token.HashRefreshToken(rawRefreshToken)
	if err := uc.refreshRepo.RevokeByHash(ctx, tokenHash, time.Now().UTC()); err != nil {
		return fmt.Errorf("revoke refresh token on logout: %w", err)
	}

	uc.log.Info("user logged out and refresh token revoked")
	return nil
}
