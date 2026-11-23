package bid

import (
	"context"
	"errors"
	"fmt"

	"bboard/internal/domain/bid"
	"bboard/internal/domain/master"
	"bboard/internal/domain/task"

	"github.com/sirupsen/logrus"
)

type CreateBidInput struct {
	TaskID        int64
	MasterID      int64
	ProposedPrice int64
	Comment       string
}

type CreateBidUseCase struct {
	bidRepo    bid.Repository
	taskRepo   task.Repository
	masterRepo master.Repository
	log        *logrus.Entry
}

func NewCreateBidUseCase(
	bidRepo bid.Repository,
	taskRepo task.Repository,
	masterRepo master.Repository,
	log *logrus.Logger,
) *CreateBidUseCase {
	return &CreateBidUseCase{
		bidRepo:    bidRepo,
		taskRepo:   taskRepo,
		masterRepo: masterRepo,
		log:        log.WithField("usecase", "bid.create"),
	}
}

func (uc *CreateBidUseCase) Execute(ctx context.Context, input CreateBidInput) (*bid.Bid, error) {
	log := uc.log.WithFields(logrus.Fields{
		"task_id":   input.TaskID,
		"master_id": input.MasterID,
	})

	// 1. Проверяем, что у пользователя есть активная анкета мастера
	profile, err := uc.masterRepo.GetByUserID(ctx, input.MasterID)
	if err != nil {
		if errors.Is(err, master.ErrProfileNotFound) {
			log.Warn("create bid rejected: user has no master profile")
			return nil, bid.ErrMasterProfileNeeded
		}
		return nil, fmt.Errorf("verify master profile: %w", err)
	}
	if !profile.IsActive {
		log.Warn("create bid rejected: master profile is inactive")
		return nil, bid.ErrMasterProfileNeeded
	}

	// 2. Проверяем существование и статус задачи
	t, err := uc.taskRepo.GetByID(ctx, input.TaskID)
	if err != nil {
		return nil, err
	}
	if t.AuthorID == input.MasterID {
		log.Warn("create bid rejected: self bid")
		return nil, bid.ErrSelfBidNotAllowed
	}
	if t.Status != task.StatusOpen {
		log.Warn("create bid rejected: task is not open")
		return nil, bid.ErrTaskNotOpenForBids
	}

	// 3. Создаём доменную сущность отклика и сохраняем
	newBid, err := bid.NewBid(input.TaskID, input.MasterID, input.ProposedPrice, input.Comment)
	if err != nil {
		log.WithError(err).Warn("create bid rejected: validation failed")
		return nil, err
	}

	if err := uc.bidRepo.Create(ctx, newBid); err != nil {
		if errors.Is(err, bid.ErrBidAlreadyExists) {
			log.Warn("create bid rejected: duplicate bid")
			return nil, err
		}
		log.WithError(err).Error("failed to persist task bid")
		return nil, fmt.Errorf("create bid: %w", err)
	}

	log.WithField("bid_id", newBid.ID).Info("task bid submitted")
	return newBid, nil
}
