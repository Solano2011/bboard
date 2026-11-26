package review

import "context"

// Repository определяет контракт хранения отзывов и пересчёта рейтинга мастера.
type Repository interface {
	Create(ctx context.Context, r *Review) error
	GetByID(ctx context.Context, id int64) (*Review, error)
	GetByTaskID(ctx context.Context, taskID int64) (*Review, error)
	GetByTaskIDs(ctx context.Context, taskIDs []int64) (map[int64]*Review, error)
	ListByMasterProfileID(ctx context.Context, masterProfileID int64, limit, offset int) ([]*Review, error)
	Update(ctx context.Context, r *Review) error
	Delete(ctx context.Context, id int64) error
}
