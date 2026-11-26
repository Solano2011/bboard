package master

import (
	"context"
	"fmt"

	"bboard/internal/domain/master"
	"bboard/internal/domain/review"

	"github.com/sirupsen/logrus"
)

type PublicProfileOutput struct {
	Profile         *master.Profile
	Services        []*master.Service
	Reviews         []*review.Review
	ContactsVisible bool
}

type GetPublicProfileUseCase struct {
	masterRepo master.Repository
	reviewRepo review.Repository
	log        *logrus.Entry
}

func NewGetPublicProfileUseCase(
	masterRepo master.Repository,
	reviewRepo review.Repository,
	log *logrus.Logger,
) *GetPublicProfileUseCase {
	return &GetPublicProfileUseCase{
		masterRepo: masterRepo,
		reviewRepo: reviewRepo,
		log:        log.WithField("usecase", "master.get_public_profile"),
	}
}

func (uc *GetPublicProfileUseCase) Execute(ctx context.Context, profileID, viewerUserID int64) (*PublicProfileOutput, error) {
	if profileID <= 0 {
		return nil, master.ErrProfileNotFound
	}

	profile, err := uc.masterRepo.GetProfileByID(ctx, profileID)
	if err != nil {
		return nil, err
	}

	contactsVisible := profile.IsContactsPublic || (viewerUserID > 0 && viewerUserID == profile.UserID)
	if !contactsVisible && viewerUserID > 0 {
		hasShared, err := uc.masterRepo.HasSharedTaskWithProfile(ctx, viewerUserID, profile.ID)
		if err != nil {
			return nil, fmt.Errorf("check shared task visibility: %w", err)
		}
		contactsVisible = hasShared
	}

	if !contactsVisible {
		profile.ContactPhone = ""
		profile.Telegram = ""
	}

	services, err := uc.masterRepo.ListServicesByProfileID(ctx, profile.ID, true)
	if err != nil {
		return nil, fmt.Errorf("list profile services: %w", err)
	}

	reviews, err := uc.reviewRepo.ListByMasterProfileID(ctx, profile.ID, 50, 0)
	if err != nil {
		return nil, fmt.Errorf("list profile reviews: %w", err)
	}

	return &PublicProfileOutput{
		Profile:         profile,
		Services:        services,
		Reviews:         reviews,
		ContactsVisible: contactsVisible,
	}, nil
}
