package master

import "context"

// ServiceListFilter задаёт параметры поиска анкет услуг мастеров в каталоге.
type ServiceListFilter struct {
	Category   string
	City       string
	OnlyActive bool
	Limit      int
	Offset     int
}

// Repository определяет контракт для работы с профилями мастеров и их анкетами услуг.
type Repository interface {
	CreateProfile(ctx context.Context, p *Profile) error
	GetProfileByID(ctx context.Context, id int64) (*Profile, error)
	GetProfileByUserID(ctx context.Context, userID int64) (*Profile, error)
	UpdateProfile(ctx context.Context, p *Profile) error
	DeleteProfile(ctx context.Context, id int64) error
	DeleteProfileIfUnused(ctx context.Context, id int64) error
	HasSharedTaskWithProfile(ctx context.Context, viewerUserID, profileID int64) (bool, error)

	CreateService(ctx context.Context, s *Service) error
	GetServiceByID(ctx context.Context, id int64) (*Service, error)
	UpdateService(ctx context.Context, s *Service) error
	DeleteService(ctx context.Context, id int64) error
	CountActiveServices(ctx context.Context, profileID int64) (int, error)
	ListServicesByProfileID(ctx context.Context, profileID int64, onlyActive bool) ([]*Service, error)
	ListCatalogServices(ctx context.Context, filter ServiceListFilter) ([]*ServiceCatalogItem, error)
}
