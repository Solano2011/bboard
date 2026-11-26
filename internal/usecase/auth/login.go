package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	domainAuth "bboard/internal/domain/auth"
	"bboard/internal/domain/user"
	"bboard/pkg/hash"

	"github.com/sirupsen/logrus"
)

// LoginInput — входные данные для сценария аутентификации.
type LoginInput struct {
	Email    string
	Password string
}

// LoginUseCase реализует бизнес-сценарий входа пользователя по email и паролю.
type LoginUseCase struct {
	userRepo    user.Repository
	refreshRepo domainAuth.RefreshTokenRepository
	tokenGen    TokenGenerator
	refreshTTL  time.Duration
	log         *logrus.Entry
}

// NewLoginUseCase собирает зависимости для сценария входа.
func NewLoginUseCase(
	userRepo user.Repository,
	refreshRepo domainAuth.RefreshTokenRepository,
	tokenGen TokenGenerator,
	refreshTTL time.Duration,
	log *logrus.Logger,
) *LoginUseCase {
	return &LoginUseCase{
		userRepo:    userRepo,
		refreshRepo: refreshRepo,
		tokenGen:    tokenGen,
		refreshTTL:  refreshTTL,
		log:         log.WithField("usecase", "auth.login"),
	}
}

// Execute проверяет учётные данные пользователя и возвращает пару access + refresh токенов.
func (uc *LoginUseCase) Execute(ctx context.Context, input LoginInput) (*AuthOutput, error) {
	normalizedEmail := strings.ToLower(strings.TrimSpace(input.Email))
	log := uc.log.WithField("email", normalizedEmail)

	if normalizedEmail == "" || strings.TrimSpace(input.Password) == "" {
		log.Warn("login rejected: empty email or password")
		return nil, user.ErrInvalidCreds
	}

	existingUser, err := uc.userRepo.GetByEmail(ctx, normalizedEmail)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			log.Warn("login rejected: user not found")
			return nil, user.ErrInvalidCreds
		}
		log.WithError(err).Error("failed to fetch user by email")
		return nil, fmt.Errorf("get user by email: %w", err)
	}

	isValid, err := hash.CheckPassword(input.Password, existingUser.PasswordHash)
	if err != nil {
		log.WithError(err).Error("failed to compare password hash")
		return nil, fmt.Errorf("check password: %w", err)
	}
	if !isValid {
		log.Warn("login rejected: password mismatch")
		return nil, user.ErrInvalidCreds
	}

	accessToken, refreshToken, err := issueTokenPair(
		ctx, existingUser.ID, uc.tokenGen, uc.refreshRepo, uc.refreshTTL,
	)
	if err != nil {
		log.WithError(err).WithField("user_id", existingUser.ID).Error("failed to issue token pair")
		return nil, err
	}

	log.WithField("user_id", existingUser.ID).Info("user successfully logged in")

	return &AuthOutput{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         existingUser,
	}, nil
}
