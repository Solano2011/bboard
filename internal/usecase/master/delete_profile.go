package master

import (
	"context"
	"fmt"

	"bboard/internal/domain/master"

	"github.com/sirupsen/logrus"
)

type DeleteProfileInput struct {
	ProfileID int64
	ActorID   int64
}

type DeleteProfileUseCase struct {
	masterRepo master.Repository
	log        *logrus.Entry
}

func NewDeleteProfileUseCase(masterRepo master.Repository, log *logrus.Logger) *DeleteProfileUseCase {
	return &DeleteProfileUseCase{
		masterRepo: masterRepo,
		log:        log.WithField("usecase", "master.delete_profile"),
	}
}

func (uc *DeleteProfileUseCase) Execute(ctx context.Context, input DeleteProfileInput) error {
	if input.ProfileID <= 0 {
		return master.ErrProfileNotFound
	}

	profile, err := uc.masterRepo.GetByID(ctx, input.ProfileID)
	if err != nil {
		return err
	}

	// Проверяем, что удаление запросил именно владелец анкеты
	if profile.UserID != input.ActorID {
		uc.log.WithFields(logrus.Fields{
			"profile_id":      input.ProfileID,
			"profile_user_id": profile.UserID,
			"actor_id":        input.ActorID,
		}).Warn("delete master profile rejected: forbidden action")
		return master.ErrForbiddenAction
	}

	if err := uc.masterRepo.Delete(ctx, profile.ID); err != nil {
		uc.log.WithError(err).Error("failed to delete master profile")
		return fmt.Errorf("delete master profile: %w", err)
	}

	uc.log.WithField("profile_id", profile.ID).Info("master profile successfully deleted")
	return nil
}
