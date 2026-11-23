package master

import "context"

// ListFilter задаёт параметры поиска анкет мастеров в каталоге.
type ListFilter struct {
	Category   string
	City       string
	OnlyActive bool
	Limit      int
	Offset     int
}

// Repository определяет контракт для работы с хранилищем анкет мастеров.
type Repository interface {
	Create(ctx context.Context, p *Profile) error
	GetByID(ctx context.Context, id int64) (*Profile, error)
	GetByUserID(ctx context.Context, userID int64) (*Profile, error)
	List(ctx context.Context, filter ListFilter) ([]*Profile, error)
	Update(ctx context.Context, p *Profile) error
}
