package master

import (
	"context"
	"fmt"

	"bboard/internal/domain/master"

	"github.com/sirupsen/logrus"
)

type ListProfilesInput struct {
	Category string
	City     string
	Limit    int
	Offset   int
}

type ListProfilesUseCase struct {
	masterRepo master.Repository
	log        *logrus.Entry
}

func NewListProfilesUseCase(masterRepo master.Repository, log *logrus.Logger) *ListProfilesUseCase {
	return &ListProfilesUseCase{
		masterRepo: masterRepo,
		log:        log.WithField("usecase", "master.list_profiles"),
	}
}

func (uc *ListProfilesUseCase) Execute(ctx context.Context, input ListProfilesInput) ([]*master.Profile, error) {
	if input.Limit <= 0 || input.Limit > 100 {
		input.Limit = 20
	}
	if input.Offset < 0 {
		input.Offset = 0
	}

	profiles, err := uc.masterRepo.List(ctx, master.ListFilter{
		Category:   input.Category,
		City:       input.City,
		OnlyActive: true,
		Limit:      input.Limit,
		Offset:     input.Offset,
	})
	if err != nil {
		uc.log.WithError(err).Error("failed to list master profiles")
		return nil, fmt.Errorf("list master profiles: %w", err)
	}

	return profiles, nil
}
