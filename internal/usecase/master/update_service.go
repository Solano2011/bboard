package master

import (
	"context"

	"bboard/internal/domain/master"

	"github.com/sirupsen/logrus"
)

type UpdateServiceInput struct {
	ServiceID             int64
	ActorUserID           int64
	Category              string
	Title                 string
	ExperienceDescription string
	Price                 int64
	IsActive              *bool
}

type UpdateServiceUseCase struct {
	masterRepo master.Repository
	log        *logrus.Entry
}

func NewUpdateServiceUseCase(masterRepo master.Repository, log *logrus.Logger) *UpdateServiceUseCase {
	return &UpdateServiceUseCase{
		masterRepo: masterRepo,
		log:        log.WithField("usecase", "master.update_service"),
	}
}

func (uc *UpdateServiceUseCase) Execute(ctx context.Context, input UpdateServiceInput) (*master.Service, error) {
	if input.ServiceID <= 0 {
		return nil, master.ErrServiceNotFound
	}

	profile, err := uc.masterRepo.GetProfileByUserID(ctx, input.ActorUserID)
	if err != nil {
		return nil, err
	}

	svc, err := uc.masterRepo.GetServiceByID(ctx, input.ServiceID)
	if err != nil {
		return nil, err
	}

	if svc.MasterProfileID != profile.ID {
		return nil, master.ErrForbiddenAction
	}

	isActive := svc.IsActive
	if input.IsActive != nil {
		isActive = *input.IsActive
	}

	if err := svc.Update(
		input.Category,
		input.Title,
		input.ExperienceDescription,
		input.Price,
		isActive,
	); err != nil {
		uc.log.WithError(err).Warn("update service rejected: validation failed")
		return nil, err
	}

	if err := uc.masterRepo.UpdateService(ctx, svc); err != nil {
		return nil, err
	}

	uc.log.WithField("service_id", svc.ID).Info("master service updated")
	return svc, nil
}
