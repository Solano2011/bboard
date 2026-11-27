package task

import (
	"context"

	"bboard/internal/domain/user"
)

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
	Update(ctx context.Context, t *Task, allowedPrevStatuses ...Status) error
	Delete(ctx context.Context, id int64, expectedStatuses ...Status) ([]string, error)
	CompleteAndIncrementMasterCounter(ctx context.Context, t *Task) error

	AddPhotos(ctx context.Context, taskID, actorID int64, actorRole user.Role, photos []*Photo) error
	ListPhotosByTaskID(ctx context.Context, taskID int64) ([]*Photo, error)
	ListPhotosByTaskIDs(ctx context.Context, taskIDs []int64) (map[int64][]*Photo, error)
	DeletePhoto(ctx context.Context, taskID, photoID int64, expectedStatuses ...Status) (*Photo, error)
}
