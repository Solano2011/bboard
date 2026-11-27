package master

import (
	"context"
	"errors"

	"bboard/internal/domain/master"
	"bboard/internal/domain/user"

	"github.com/sirupsen/logrus"
)

type DeleteServiceInput struct {
	ServiceID   int64
	ActorUserID int64
	ActorRole   user.Role
}

type DeleteServiceUseCase struct {
	masterRepo master.Repository
	log        *logrus.Entry
}

func NewDeleteServiceUseCase(masterRepo master.Repository, log *logrus.Logger) *DeleteServiceUseCase {
	return &DeleteServiceUseCase{
		masterRepo: masterRepo,
		log:        log.WithField("usecase", "master.delete_service"),
	}
}

func (uc *DeleteServiceUseCase) Execute(ctx context.Context, input DeleteServiceInput) error {
	if input.ServiceID <= 0 {
		return master.ErrServiceNotFound
	}

	svc, err := uc.masterRepo.GetServiceByID(ctx, input.ServiceID)
	if err != nil {
		return err
	}

	if !input.ActorRole.IsModeratorOrAdmin() {
		profile, err := uc.masterRepo.GetProfileByUserID(ctx, input.ActorUserID)
		if err != nil {
			if errors.Is(err, master.ErrProfileNotFound) {
				return master.ErrForbiddenAction
			}
			return err
		}
		if svc.MasterProfileID != profile.ID {
			return master.ErrForbiddenAction
		}
	}

	if err := uc.masterRepo.DeleteService(ctx, svc.ID); err != nil {
		return err
	}

	uc.log.WithFields(logrus.Fields{
		"service_id": svc.ID,
		"actor_id":   input.ActorUserID,
		"actor_role": input.ActorRole,
	}).Info("master service deleted")
	return nil
}
