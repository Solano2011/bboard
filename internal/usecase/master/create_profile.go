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

type CreateProfileInput struct {
	UserID          int64
	ContactPhone    string
	Headline        string
	Bio             string
	Category        string
	City            string
	ExperienceYears int
	MinPrice        int64
}

type CreateProfileUseCase struct {
	masterRepo master.Repository
	userRepo   user.Repository
	log        *logrus.Entry
}

func NewCreateProfileUseCase(
	masterRepo master.Repository,
	userRepo user.Repository,
	log *logrus.Logger,
) *CreateProfileUseCase {
	return &CreateProfileUseCase{
		masterRepo: masterRepo,
		userRepo:   userRepo,
		log:        log.WithField("usecase", "master.create_profile"),
	}
}

func (uc *CreateProfileUseCase) Execute(ctx context.Context, input CreateProfileInput) (*master.Profile, error) {
	log := uc.log.WithFields(logrus.Fields{
		"user_id":  input.UserID,
		"category": input.Category,
	})

	u, err := uc.userRepo.GetByID(ctx, input.UserID)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			return nil, master.ErrInvalidUserID
		}
		return nil, fmt.Errorf("verify user: %w", err)
	}

	phone := strings.TrimSpace(input.ContactPhone)
	if phone == "" {
		phone = u.Phone
	}

	profile, err := master.NewProfile(
		input.UserID,
		phone,
		input.Headline,
		input.Bio,
		input.Category,
		input.City,
		input.ExperienceYears,
		input.MinPrice,
	)
	if err != nil {
		log.WithError(err).Warn("create master profile rejected: validation failed")
		return nil, err
	}

	if err := uc.masterRepo.Create(ctx, profile); err != nil {
		if errors.Is(err, master.ErrProfileAlreadyExists) {
			log.Warn("create master profile rejected: already exists")
			return nil, err
		}
		log.WithError(err).Error("failed to persist master profile")
		return nil, fmt.Errorf("create master profile: %w", err)
	}

	profile.MasterName = u.Name
	log.WithField("profile_id", profile.ID).Info("master profile created")
	return profile, nil
}
