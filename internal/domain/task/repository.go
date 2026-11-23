package task

import "context"

// ListFilter задаёт параметры поиска и пагинации для ленты объявлений.
type ListFilter struct {
	Category string
	City     string
	Status   Status
	Limit    int
	Offset   int
}

// Repository определяет контракт для работы с хранилищем объявлений заказчиков.
type Repository interface {
	Create(ctx context.Context, t *Task) error
	GetByID(ctx context.Context, id int64) (*Task, error)
	List(ctx context.Context, filter ListFilter) ([]*Task, error)
	Update(ctx context.Context, t *Task) error
}
