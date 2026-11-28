package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	domainAuth "bboard/internal/domain/auth"
	"bboard/internal/domain/user"
	"bboard/pkg/token"

	"github.com/sirupsen/logrus"
)

// SendRegisterCodeInput — входные данные для отправки кода подтверждения при регистрации.
type SendRegisterCodeInput struct {
	Email string
}

// SendRegisterCodeUseCase генерирует и отправляет одноразовый 6-значный код подтверждения email.
type SendRegisterCodeUseCase struct {
	userRepo         user.Repository
	verificationRepo domainAuth.EmailVerificationRepository
	emailSender      domainAuth.EmailSender
	log              *logrus.Entry
}

// NewSendRegisterCodeUseCase создаёт экземпляр SendRegisterCodeUseCase.
func NewSendRegisterCodeUseCase(
	userRepo user.Repository,
	verificationRepo domainAuth.EmailVerificationRepository,
	emailSender domainAuth.EmailSender,
	log *logrus.Logger,
) *SendRegisterCodeUseCase {
	return &SendRegisterCodeUseCase{
		userRepo:         userRepo,
		verificationRepo: verificationRepo,
		emailSender:      emailSender,
		log:              log.WithField("usecase", "auth.send_register_code"),
	}
}

// Execute проверяет свободный email, сохраняет хэш кода с блокировкой cooldown и отправляет письмо до коммита.
func (uc *SendRegisterCodeUseCase) Execute(ctx context.Context, input SendRegisterCodeInput) error {
	normalizedEmail, err := user.NormalizeAndValidateEmail(input.Email)
	if err != nil {
		return err
	}

	log := uc.log.WithField("email", normalizedEmail)

	existingUser, err := uc.userRepo.GetByEmail(ctx, normalizedEmail)
	if err != nil && !errors.Is(err, user.ErrUserNotFound) {
		log.WithError(err).Error("failed to check existing user before sending code")
		return fmt.Errorf("check existing user: %w", err)
	}
	if existingUser != nil {
		log.Warn("verification code rejected: email already registered")
		return user.ErrUserAlreadyExist
	}

	code, err := token.GenerateVerificationCode()
	if err != nil {
		log.WithError(err).Error("failed to generate verification code")
		return fmt.Errorf("generate verification code: %w", err)
	}

	codeHash := token.HashVerificationCode(code)
	now := time.Now().UTC()
	expiresAt := now.Add(domainAuth.VerificationCodeTTL)

	err = uc.verificationRepo.SaveAndSendCode(
		ctx,
		normalizedEmail,
		codeHash,
		now,
		expiresAt,
		domainAuth.MaxVerificationAttempts,
		func() error {
			if sendErr := uc.emailSender.SendVerificationCode(ctx, normalizedEmail, code); sendErr != nil {
				return fmt.Errorf("%w: %w", domainAuth.ErrEmailDeliveryFailed, sendErr)
			}
			return nil
		},
	)
	if err != nil {
		if errors.Is(err, domainAuth.ErrVerificationResendTooSoon) {
			log.WithError(err).Warn("verification code resend blocked by cooldown")
			return err
		}
		log.WithError(err).Error("failed to save or send verification code")
		return err
	}

	log.Info("verification code sent successfully")
	return nil
}
