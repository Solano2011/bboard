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
	ActorUserID   int64
	ProposedPrice *int64
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
		"task_id":       input.TaskID,
		"actor_user_id": input.ActorUserID,
	})

	// 1. Проверяем, что у пользователя создан профиль мастера и есть хотя бы 1 активная услуга
	profile, err := uc.masterRepo.GetProfileByUserID(ctx, input.ActorUserID)
	if err != nil {
		if errors.Is(err, master.ErrProfileNotFound) {
			log.Warn("create response rejected: user has no master profile")
			return nil, bid.ErrMasterProfileNeeded
		}
		return nil, fmt.Errorf("verify master profile: %w", err)
	}

	activeServicesCount, err := uc.masterRepo.CountActiveServices(ctx, profile.ID)
	if err != nil {
		return nil, fmt.Errorf("count active services: %w", err)
	}
	if activeServicesCount == 0 {
		log.Warn("create response rejected: master has no active services")
		return nil, bid.ErrMasterProfileNeeded
	}

	// 2. Проверяем существование и статус задачи
	t, err := uc.taskRepo.GetByID(ctx, input.TaskID)
	if err != nil {
		return nil, err
	}
	if t.AuthorID == input.ActorUserID {
		log.Warn("create response rejected: self bid")
		return nil, bid.ErrSelfBidNotAllowed
	}
	if t.Status != task.StatusOpen {
		log.Warn("create response rejected: task is not open")
		return nil, bid.ErrTaskNotOpenForBids
	}

	// 3. Создаём доменную сущность отклика и сохраняем
	newBid, err := bid.NewBid(input.TaskID, profile.ID, input.ProposedPrice, input.Comment)
	if err != nil {
		log.WithError(err).Warn("create response rejected: validation failed")
		return nil, err
	}

	if err := uc.bidRepo.Create(ctx, newBid); err != nil {
		if errors.Is(err, bid.ErrBidAlreadyExists) {
			log.Warn("create response rejected: duplicate response")
			return nil, err
		}
		log.WithError(err).Error("failed to save task response")
		return nil, fmt.Errorf("create task response: %w", err)
	}

	newBid.MasterDisplayName = profile.DisplayName
	newBid.MasterAvatarURL = profile.AvatarURL
	newBid.MasterRatingAvg = profile.RatingAvg
	newBid.MasterReviewsCount = profile.ReviewsCount
	newBid.MasterCompletedOrdersCount = profile.CompletedOrdersCount

	log.WithFields(logrus.Fields{
		"response_id":       newBid.ID,
		"master_profile_id": profile.ID,
	}).Info("task response created")
	return newBid, nil
}
