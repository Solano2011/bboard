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

const minPasswordLength = 8

// RegisterInput — входные данные для сценария регистрации.
// Отделяем их от HTTP DTO, чтобы юзкейс можно было вызывать из любого транспорта (REST, gRPC, CLI).
type RegisterInput struct {
	Name     string
	Email    string
	Phone    string
	Password string
}

// RegisterUseCase реализует бизнес-сценарий регистрации нового пользователя.
type RegisterUseCase struct {
	userRepo user.Repository
	log      *logrus.Entry
}

// NewRegisterUseCase собирает зависимости для сценария регистрации.
func NewRegisterUseCase(userRepo user.Repository, log *logrus.Logger) *RegisterUseCase {
	return &RegisterUseCase{
		userRepo: userRepo,
		// Добавляем постоянное поле компонента для удобной фильтрации логов
		log: log.WithField("usecase", "auth.register"),
	}
}

// Execute выполняет регистрацию пользователя.
func (uc *RegisterUseCase) Execute(ctx context.Context, input RegisterInput) (*user.User, error) {
	normalizedEmail := strings.ToLower(strings.TrimSpace(input.Email))
	log := uc.log.WithField("email", normalizedEmail)

	// 1. Проверяем длину сырого пароля до хэширования (bcrypt обрезает пароли длиннее 72 байт,
	// а слишком короткие нужно отсекать сразу).
	if len(strings.TrimSpace(input.Password)) < minPasswordLength {
		log.Warn("registration rejected: weak password")
		return nil, user.ErrWeakPassword
	}

	// 2. Проверяем, не занят ли уже этот email.
	existingUser, err := uc.userRepo.GetByEmail(ctx, normalizedEmail)
	if err != nil && !errors.Is(err, user.ErrUserNotFound) {
		log.WithError(err).Error("failed to check existing user by email")
		return nil, fmt.Errorf("check existing user: %w", err)
	}
	if existingUser != nil {
		log.Warn("registration rejected: email already exists")
		return nil, user.ErrUserAlreadyExist
	}

	// 3. Хэшируем пароль через bcrypt (cost = 0 применит bcrypt.DefaultCost).
	passwordHash, err := hash.HashPassword(input.Password, 0)
	if err != nil {
		log.WithError(err).Error("failed to hash password")
		return nil, fmt.Errorf("hash password: %w", err)
	}

	// 4. Создаём валидную доменную сущность (внутри проверяются имя и формат email).
	newUser, err := user.NewUser(input.Name, normalizedEmail, input.Phone, passwordHash)
	if err != nil {
		log.WithError(err).Warn("registration rejected: domain validation failed")
		return nil, err
	}

	// 5. Сохраняем пользователя в хранилище через интерфейс репозитория.
	if err := uc.userRepo.Create(ctx, newUser); err != nil {
		// На случай гонки запросов (race condition), если два запроса пришли одновременно
		if errors.Is(err, user.ErrUserAlreadyExist) {
			log.Warn("registration rejected on save: email already exists")
			return nil, user.ErrUserAlreadyExist
		}
		log.WithError(err).Error("failed to persist new user")
		return nil, fmt.Errorf("create user in repository: %w", err)
	}

	log.WithField("user_id", newUser.ID).Info("user successfully registered")
	return newUser, nil
}
