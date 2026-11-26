package master

import (
	"context"

	"bboard/internal/domain/master"

	"github.com/sirupsen/logrus"
)

type CreateServiceInput struct {
	ActorUserID           int64
	Category              string
	Title                 string
	ExperienceDescription string
	Price                 int64
}

type CreateServiceUseCase struct {
	masterRepo master.Repository
	log        *logrus.Entry
}

func NewCreateServiceUseCase(masterRepo master.Repository, log *logrus.Logger) *CreateServiceUseCase {
	return &CreateServiceUseCase{
		masterRepo: masterRepo,
		log:        log.WithField("usecase", "master.create_service"),
	}
}

func (uc *CreateServiceUseCase) Execute(ctx context.Context, input CreateServiceInput) (*master.Service, error) {
	profile, err := uc.masterRepo.GetProfileByUserID(ctx, input.ActorUserID)
	if err != nil {
		return nil, err
	}

	svc, err := master.NewService(
		profile.ID,
		input.Category,
		input.Title,
		input.ExperienceDescription,
		input.Price,
	)
	if err != nil {
		uc.log.WithError(err).Warn("create service rejected: validation failed")
		return nil, err
	}

	if err := uc.masterRepo.CreateService(ctx, svc); err != nil {
		uc.log.WithError(err).Warn("create service failed")
		return nil, err
	}

	uc.log.WithFields(logrus.Fields{
		"service_id": svc.ID,
		"profile_id": profile.ID,
		"category":   svc.Category,
	}).Info("master service created")
	return svc, nil
}
