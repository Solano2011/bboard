package review

import (
	"context"
	"fmt"

	"bboard/internal/domain/review"
	"bboard/internal/domain/task"

	"github.com/sirupsen/logrus"
)

type CreateReviewInput struct {
	TaskID      int64
	ActorUserID int64
	Rating      int
	Comment     string
}

type CreateReviewUseCase struct {
	reviewRepo review.Repository
	taskRepo   task.Repository
	log        *logrus.Entry
}

func NewCreateReviewUseCase(
	reviewRepo review.Repository,
	taskRepo task.Repository,
	log *logrus.Logger,
) *CreateReviewUseCase {
	return &CreateReviewUseCase{
		reviewRepo: reviewRepo,
		taskRepo:   taskRepo,
		log:        log.WithField("usecase", "review.create"),
	}
}

func (uc *CreateReviewUseCase) Execute(ctx context.Context, input CreateReviewInput) (*review.Review, error) {
	t, err := uc.taskRepo.GetByID(ctx, input.TaskID)
	if err != nil {
		return nil, err
	}

	if t.AuthorID != input.ActorUserID {
		return nil, review.ErrForbiddenReview
	}
	if t.Status != task.StatusCompleted || t.AssignedMasterID == nil {
		return nil, review.ErrTaskNotCompleted
	}

	rev, err := review.NewReview(t.ID, *t.AssignedMasterID, input.ActorUserID, input.Rating, input.Comment)
	if err != nil {
		return nil, err
	}

	if err := uc.reviewRepo.Create(ctx, rev); err != nil {
		return nil, err
	}

	created, err := uc.reviewRepo.GetByID(ctx, rev.ID)
	if err != nil {
		return nil, fmt.Errorf("reload created review: %w", err)
	}

	uc.log.WithFields(logrus.Fields{
		"review_id":         created.ID,
		"task_id":           created.TaskID,
		"master_profile_id": created.MasterProfileID,
	}).Info("master review created")
	return created, nil
}
