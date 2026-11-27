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

const minPasswordLength = 8

// RegisterInput — входные данные для сценария регистрации.
type RegisterInput struct {
	Name     string
	Email    string
	Phone    string
	Password string
}

// RegisterUseCase реализует бизнес-сценарий регистрации нового пользователя.
type RegisterUseCase struct {
	userRepo    user.Repository
	refreshRepo domainAuth.RefreshTokenRepository
	tokenGen    TokenGenerator
	refreshTTL  time.Duration
	adminEmail  string
	log         *logrus.Entry
}

// NewRegisterUseCase собирает зависимости для сценария регистрации.
func NewRegisterUseCase(
	userRepo user.Repository,
	refreshRepo domainAuth.RefreshTokenRepository,
	tokenGen TokenGenerator,
	refreshTTL time.Duration,
	adminEmail string,
	log *logrus.Logger,
) *RegisterUseCase {
	return &RegisterUseCase{
		userRepo:    userRepo,
		refreshRepo: refreshRepo,
		tokenGen:    tokenGen,
		refreshTTL:  refreshTTL,
		adminEmail:  strings.ToLower(strings.TrimSpace(adminEmail)),
		log:         log.WithField("usecase", "auth.register"),
	}
}

// Execute регистрирует пользователя и сразу выпускает пару access + refresh токенов.
func (uc *RegisterUseCase) Execute(ctx context.Context, input RegisterInput) (*AuthOutput, error) {
	normalizedEmail := strings.ToLower(strings.TrimSpace(input.Email))
	log := uc.log.WithField("email", normalizedEmail)

	if len(strings.TrimSpace(input.Password)) < minPasswordLength {
		log.Warn("registration rejected: weak password")
		return nil, user.ErrWeakPassword
	}

	existingUser, err := uc.userRepo.GetByEmail(ctx, normalizedEmail)
	if err != nil && !errors.Is(err, user.ErrUserNotFound) {
		log.WithError(err).Error("failed to check existing user by email")
		return nil, fmt.Errorf("check existing user: %w", err)
	}
	if existingUser != nil {
		log.Warn("registration rejected: email already exists")
		return nil, user.ErrUserAlreadyExist
	}

	passwordHash, err := hash.HashPassword(input.Password, 0)
	if err != nil {
		log.WithError(err).Error("failed to hash password")
		return nil, fmt.Errorf("hash password: %w", err)
	}

	newUser, err := user.NewUser(input.Name, normalizedEmail, input.Phone, passwordHash)
	if err != nil {
		log.WithError(err).Warn("registration rejected: domain validation failed")
		return nil, err
	}
	if uc.adminEmail != "" && newUser.Email == uc.adminEmail {
		newUser.Role = user.RoleAdmin
	}

	if err := uc.userRepo.Create(ctx, newUser); err != nil {
		if errors.Is(err, user.ErrUserAlreadyExist) {
			log.Warn("registration rejected on save: email already exists")
			return nil, user.ErrUserAlreadyExist
		}
		log.WithError(err).Error("failed to persist new user")
		return nil, fmt.Errorf("create user in repository: %w", err)
	}

	accessToken, refreshToken, err := issueTokenPair(
		ctx, newUser.ID, uc.tokenGen, uc.refreshRepo, uc.refreshTTL,
	)
	if err != nil {
		log.WithError(err).WithField("user_id", newUser.ID).Error("failed to issue token pair on register")
		return nil, err
	}

	log.WithField("user_id", newUser.ID).Info("user successfully registered")

	return &AuthOutput{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         newUser,
	}, nil
}
