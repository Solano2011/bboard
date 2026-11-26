package master

import (
	"context"

	"bboard/internal/domain/master"

	"github.com/sirupsen/logrus"
)

type DeleteServiceInput struct {
	ServiceID   int64
	ActorUserID int64
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

	profile, err := uc.masterRepo.GetProfileByUserID(ctx, input.ActorUserID)
	if err != nil {
		return err
	}

	svc, err := uc.masterRepo.GetServiceByID(ctx, input.ServiceID)
	if err != nil {
		return err
	}

	if svc.MasterProfileID != profile.ID {
		return master.ErrForbiddenAction
	}

	if err := uc.masterRepo.DeleteService(ctx, svc.ID); err != nil {
		return err
	}

	uc.log.WithField("service_id", svc.ID).Info("master service deleted")
	return nil
}
