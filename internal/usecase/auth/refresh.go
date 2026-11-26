package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	domainAuth "bboard/internal/domain/auth"
	"bboard/internal/domain/user"
	"bboard/pkg/token"

	"github.com/sirupsen/logrus"
)

type RefreshUseCase struct {
	userRepo    user.Repository
	refreshRepo domainAuth.RefreshTokenRepository
	tokenGen    TokenGenerator
	refreshTTL  time.Duration
	log         *logrus.Entry
}

func NewRefreshUseCase(
	userRepo user.Repository,
	refreshRepo domainAuth.RefreshTokenRepository,
	tokenGen TokenGenerator,
	refreshTTL time.Duration,
	log *logrus.Logger,
) *RefreshUseCase {
	return &RefreshUseCase{
		userRepo:    userRepo,
		refreshRepo: refreshRepo,
		tokenGen:    tokenGen,
		refreshTTL:  refreshTTL,
		log:         log.WithField("usecase", "auth.refresh"),
	}
}

func (uc *RefreshUseCase) Execute(ctx context.Context, rawRefreshToken string) (*AuthOutput, error) {
	if strings.TrimSpace(rawRefreshToken) == "" {
		return nil, domainAuth.ErrInvalidRefreshToken
	}

	now := time.Now().UTC()
	tokenHash := token.HashRefreshToken(rawRefreshToken)

	rt, err := uc.refreshRepo.GetByHash(ctx, tokenHash)
	if err != nil {
		return nil, err
	}

	// Повторное использование уже отозванного токена — признак компрометации: отзываем все сессии пользователя.
	if rt.IsRevoked() {
		uc.log.WithField("user_id", rt.UserID).Warn("refresh token reuse detected: revoking all user sessions")
		if revokeErr := uc.refreshRepo.RevokeAllByUserID(ctx, rt.UserID, now); revokeErr != nil {
			uc.log.WithError(revokeErr).Error("failed to revoke all user sessions")
		}
		return nil, domainAuth.ErrRefreshTokenRevoked
	}

	if rt.IsExpired(now) {
		_, _ = uc.refreshRepo.RevokeIfActive(ctx, rt.ID, now)
		return nil, domainAuth.ErrRefreshTokenExpired
	}

	// Атомарная ротация: если параллельный запрос уже отозвал этот токен, также отзываем все сессии.
	revoked, err := uc.refreshRepo.RevokeIfActive(ctx, rt.ID, now)
	if err != nil {
		return nil, fmt.Errorf("revoke old refresh token: %w", err)
	}
	if !revoked {
		uc.log.WithField("user_id", rt.UserID).Warn("concurrent refresh token reuse detected: revoking all user sessions")
		_ = uc.refreshRepo.RevokeAllByUserID(ctx, rt.UserID, now)
		return nil, domainAuth.ErrRefreshTokenRevoked
	}

	u, err := uc.userRepo.GetByID(ctx, rt.UserID)
	if err != nil {
		return nil, fmt.Errorf("get user for refresh: %w", err)
	}

	accessToken, newRefreshToken, err := issueTokenPair(ctx, u.ID, uc.tokenGen, uc.refreshRepo, uc.refreshTTL)
	if err != nil {
		return nil, err
	}

	uc.log.WithField("user_id", u.ID).Info("token pair successfully rotated")

	return &AuthOutput{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		User:         u,
	}, nil
}
