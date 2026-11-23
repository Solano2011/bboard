package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"bboard/internal/domain/user"
	"bboard/pkg/hash"

	"github.com/sirupsen/logrus"
)

// TokenGenerator описывает контракт для выпуска токенов доступа (например, JWT).
// Реализация будет лежать в pkg/token или internal/infrastructure и подставляться через DI.
type TokenGenerator interface {
	GenerateToken(userID int64) (string, error)
}

// LoginInput — входные данные для сценария аутентификации.
type LoginInput struct {
	Email    string
	Password string
}

// LoginOutput — результат успешного входа.
type LoginOutput struct {
	Token string
	User  *user.User
}

// LoginUseCase реализует бизнес-сценарий входа пользователя по email и паролю.
type LoginUseCase struct {
	userRepo user.Repository
	tokenGen TokenGenerator
	log      *logrus.Entry
}

// NewLoginUseCase собирает зависимости для сценария входа.
func NewLoginUseCase(userRepo user.Repository, tokenGen TokenGenerator, log *logrus.Logger) *LoginUseCase {
	return &LoginUseCase{
		userRepo: userRepo,
		tokenGen: tokenGen,
		log:      log.WithField("usecase", "auth.login"),
	}
}

// Execute проверяет учётные данные пользователя и возвращает токен доступа.
func (uc *LoginUseCase) Execute(ctx context.Context, input LoginInput) (*LoginOutput, error) {
	normalizedEmail := strings.ToLower(strings.TrimSpace(input.Email))
	log := uc.log.WithField("email", normalizedEmail)

	if normalizedEmail == "" || strings.TrimSpace(input.Password) == "" {
		log.Warn("login rejected: empty email or password")
		return nil, user.ErrInvalidCreds
	}

	// 1. Ищем пользователя по email.
	// Важно: если пользователь не найден, возвращаем общую ошибку ErrInvalidCreds,
	// чтобы злоумышленник не мог перебором узнавать, какие email зарегистрированы в системе.
	existingUser, err := uc.userRepo.GetByEmail(ctx, normalizedEmail)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			log.Warn("login rejected: user not found")
			return nil, user.ErrInvalidCreds
		}
		log.WithError(err).Error("failed to fetch user by email")
		return nil, fmt.Errorf("get user by email: %w", err)
	}

	// 2. Сравниваем переданный пароль с bcrypt-хэшем из базы.
	isValid, err := hash.CheckPassword(input.Password, existingUser.PasswordHash)
	if err != nil {
		log.WithError(err).Error("failed to compare password hash")
		return nil, fmt.Errorf("check password: %w", err)
	}
	if !isValid {
		log.Warn("login rejected: password mismatch")
		return nil, user.ErrInvalidCreds
	}

	// 3. Генерируем токен доступа для найденного user.ID.
	token, err := uc.tokenGen.GenerateToken(existingUser.ID)
	if err != nil {
		log.WithError(err).WithField("user_id", existingUser.ID).Error("failed to generate access token")
		return nil, fmt.Errorf("generate token: %w", err)
	}

	log.WithField("user_id", existingUser.ID).Info("user successfully logged in")

	return &LoginOutput{
		Token: token,
		User:  existingUser,
	}, nil
}
