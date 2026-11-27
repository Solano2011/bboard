package master

import (
	"context"
	"errors"

	"bboard/internal/domain/master"
	"bboard/internal/domain/user"

	"github.com/sirupsen/logrus"
)

type UpdateServiceInput struct {
	ServiceID             int64
	ActorUserID           int64
	ActorRole             user.Role
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

	svc, err := uc.masterRepo.GetServiceByID(ctx, input.ServiceID)
	if err != nil {
		return nil, err
	}

	if !input.ActorRole.IsModeratorOrAdmin() {
		profile, err := uc.masterRepo.GetProfileByUserID(ctx, input.ActorUserID)
		if err != nil {
			if errors.Is(err, master.ErrProfileNotFound) {
				return nil, master.ErrForbiddenAction
			}
			return nil, err
		}
		if svc.MasterProfileID != profile.ID {
			return nil, master.ErrForbiddenAction
		}
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

	uc.log.WithFields(logrus.Fields{
		"service_id": svc.ID,
		"actor_id":   input.ActorUserID,
		"actor_role": input.ActorRole,
	}).Info("master service updated")
	return svc, nil
}
