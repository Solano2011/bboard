package master

import (
	"context"

	"bboard/internal/domain/master"

	"github.com/sirupsen/logrus"
)

type ListServicesInput struct {
	Category string
	City     string
	Limit    int
	Offset   int
}

type ListServicesUseCase struct {
	masterRepo master.Repository
	log        *logrus.Entry
}

func NewListServicesUseCase(masterRepo master.Repository, log *logrus.Logger) *ListServicesUseCase {
	return &ListServicesUseCase{
		masterRepo: masterRepo,
		log:        log.WithField("usecase", "master.list_services"),
	}
}

func (uc *ListServicesUseCase) Execute(ctx context.Context, input ListServicesInput) ([]*master.ServiceCatalogItem, error) {
	limit := input.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := input.Offset
	if offset < 0 {
		offset = 0
	}

	return uc.masterRepo.ListCatalogServices(ctx, master.ServiceListFilter{
		Category:   input.Category,
		City:       input.City,
		OnlyActive: true,
		Limit:      limit,
		Offset:     offset,
	})
}
