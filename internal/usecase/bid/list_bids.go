package bid

import (
	"context"
	"fmt"

	"bboard/internal/domain/bid"
	"bboard/internal/domain/task"
	"bboard/internal/domain/user"

	"github.com/sirupsen/logrus"
)

type ListBidsInput struct {
	TaskID    int64
	ActorID   int64
	ActorRole user.Role
}

type ListBidsUseCase struct {
	bidRepo  bid.Repository
	taskRepo task.Repository
	log      *logrus.Entry
}

func NewListBidsUseCase(
	bidRepo bid.Repository,
	taskRepo task.Repository,
	log *logrus.Logger,
) *ListBidsUseCase {
	return &ListBidsUseCase{
		bidRepo:  bidRepo,
		taskRepo: taskRepo,
		log:      log.WithField("usecase", "bid.list"),
	}
}

func (uc *ListBidsUseCase) Execute(ctx context.Context, input ListBidsInput) ([]*bid.Bid, error) {
	t, err := uc.taskRepo.GetByID(ctx, input.TaskID)
	if err != nil {
		return nil, err
	}

	// Смотреть отклики может автор задачи или модератор/администратор
	if t.AuthorID != input.ActorID && !input.ActorRole.IsModeratorOrAdmin() {
		uc.log.WithFields(logrus.Fields{
			"task_id":    input.TaskID,
			"actor_id":   input.ActorID,
			"actor_role": input.ActorRole,
		}).Warn("list bids rejected: forbidden")
		return nil, bid.ErrForbiddenView
	}

	bids, err := uc.bidRepo.ListByTaskID(ctx, input.TaskID)
	if err != nil {
		uc.log.WithError(err).Error("failed to list task bids")
		return nil, fmt.Errorf("list task bids: %w", err)
	}

	return bids, nil
}
