package master

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"bboard/internal/domain/master"
	"bboard/internal/domain/user"

	"github.com/sirupsen/logrus"
)

type UpsertProfileInput struct {
	UserID           int64
	DisplayName      string
	AvatarURL        string
	About            string
	City             string
	District         string
	ContactPhone     string
	Telegram         string
	IsContactsPublic *bool
}

type UpsertProfileUseCase struct {
	masterRepo master.Repository
	userRepo   user.Repository
	log        *logrus.Entry
}

func NewUpsertProfileUseCase(
	masterRepo master.Repository,
	userRepo user.Repository,
	log *logrus.Logger,
) *UpsertProfileUseCase {
	return &UpsertProfileUseCase{
		masterRepo: masterRepo,
		userRepo:   userRepo,
		log:        log.WithField("usecase", "master.upsert_profile"),
	}
}

func (uc *UpsertProfileUseCase) Execute(ctx context.Context, input UpsertProfileInput) (*master.Profile, error) {
	u, err := uc.userRepo.GetByID(ctx, input.UserID)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			return nil, master.ErrInvalidUserID
		}
		return nil, fmt.Errorf("verify user: %w", err)
	}

	displayName := strings.TrimSpace(input.DisplayName)
	if displayName == "" {
		displayName = u.Name
	}

	existing, err := uc.masterRepo.GetProfileByUserID(ctx, input.UserID)
	if err != nil && !errors.Is(err, master.ErrProfileNotFound) {
		return nil, fmt.Errorf("get existing master profile: %w", err)
	}

	if existing == nil {
		isContactsPublic := true
		if input.IsContactsPublic != nil {
			isContactsPublic = *input.IsContactsPublic
		}

		profile, err := master.NewProfile(
			input.UserID,
			displayName,
			input.AvatarURL,
			input.About,
			input.City,
			input.District,
			input.ContactPhone,
			input.Telegram,
			isContactsPublic,
		)
		if err != nil {
			uc.log.WithError(err).Warn("create master profile rejected: validation failed")
			return nil, err
		}

		if err := uc.masterRepo.CreateProfile(ctx, profile); err != nil {
			return nil, err
		}

		uc.log.WithField("profile_id", profile.ID).Info("master profile created")
		return profile, nil
	}

	isContactsPublic := existing.IsContactsPublic
	if input.IsContactsPublic != nil {
		isContactsPublic = *input.IsContactsPublic
	}

	if err := existing.Update(
		input.UserID,
		displayName,
		input.AvatarURL,
		input.About,
		input.City,
		input.District,
		input.ContactPhone,
		input.Telegram,
		isContactsPublic,
	); err != nil {
		uc.log.WithError(err).Warn("update master profile rejected: validation failed")
		return nil, err
	}

	if err := uc.masterRepo.UpdateProfile(ctx, existing); err != nil {
		return nil, err
	}

	uc.log.WithField("profile_id", existing.ID).Info("master profile updated")
	return existing, nil
}
