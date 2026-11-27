package master

import (
	"context"

	"bboard/internal/domain/master"

	"github.com/sirupsen/logrus"
)

type DeleteMyProfileUseCase struct {
	masterRepo master.Repository
	log        *logrus.Entry
}

func NewDeleteMyProfileUseCase(masterRepo master.Repository, log *logrus.Logger) *DeleteMyProfileUseCase {
	return &DeleteMyProfileUseCase{
		masterRepo: masterRepo,
		log:        log.WithField("usecase", "master.delete_my_profile"),
	}
}

func (uc *DeleteMyProfileUseCase) Execute(ctx context.Context, actorUserID int64) error {
	if actorUserID <= 0 {
		return master.ErrInvalidUserID
	}

	profile, err := uc.masterRepo.GetProfileByUserID(ctx, actorUserID)
	if err != nil {
		return err
	}

	if err := uc.masterRepo.DeleteProfileIfUnused(ctx, profile.ID); err != nil {
		uc.log.WithFields(logrus.Fields{
			"profile_id": profile.ID,
			"user_id":    actorUserID,
		}).WithError(err).Warn("delete my master profile rejected")
		return err
	}

	uc.log.WithFields(logrus.Fields{
		"profile_id": profile.ID,
		"user_id":    actorUserID,
	}).Info("master profile deleted by owner")
	return nil
}
