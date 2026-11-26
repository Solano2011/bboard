package master

import (
	"context"
	"fmt"

	"bboard/internal/domain/master"

	"github.com/sirupsen/logrus"
)

type MyProfileOutput struct {
	Profile  *master.Profile
	Services []*master.Service
}

type GetMyProfileUseCase struct {
	masterRepo master.Repository
	log        *logrus.Entry
}

func NewGetMyProfileUseCase(masterRepo master.Repository, log *logrus.Logger) *GetMyProfileUseCase {
	return &GetMyProfileUseCase{
		masterRepo: masterRepo,
		log:        log.WithField("usecase", "master.get_my_profile"),
	}
}

func (uc *GetMyProfileUseCase) Execute(ctx context.Context, userID int64) (*MyProfileOutput, error) {
	if userID <= 0 {
		return nil, master.ErrInvalidUserID
	}

	profile, err := uc.masterRepo.GetProfileByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	services, err := uc.masterRepo.ListServicesByProfileID(ctx, profile.ID, false)
	if err != nil {
		return nil, fmt.Errorf("list my services: %w", err)
	}

	return &MyProfileOutput{
		Profile:  profile,
		Services: services,
	}, nil
}
