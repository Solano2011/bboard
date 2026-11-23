package bid

import "context"

type Repository interface {
	Create(ctx context.Context, b *Bid) error
	ListByTaskID(ctx context.Context, taskID int64) ([]*Bid, error)
}
