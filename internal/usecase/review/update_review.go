package review

import (
	"context"
	"fmt"

	"bboard/internal/domain/review"

	"github.com/sirupsen/logrus"
)

type UpdateReviewInput struct {
	ReviewID    int64
	ActorUserID int64
	Rating      int
	Comment     string
}

type UpdateReviewUseCase struct {
	reviewRepo review.Repository
	log        *logrus.Entry
}

func NewUpdateReviewUseCase(reviewRepo review.Repository, log *logrus.Logger) *UpdateReviewUseCase {
	return &UpdateReviewUseCase{
		reviewRepo: reviewRepo,
		log:        log.WithField("usecase", "review.update"),
	}
}

func (uc *UpdateReviewUseCase) Execute(ctx context.Context, input UpdateReviewInput) (*review.Review, error) {
	if input.ReviewID <= 0 {
		return nil, review.ErrReviewNotFound
	}

	rev, err := uc.reviewRepo.GetByID(ctx, input.ReviewID)
	if err != nil {
		return nil, err
	}

	if err := rev.Update(input.ActorUserID, input.Rating, input.Comment); err != nil {
		return nil, err
	}

	if err := uc.reviewRepo.Update(ctx, rev); err != nil {
		return nil, err
	}

	updated, err := uc.reviewRepo.GetByID(ctx, rev.ID)
	if err != nil {
		return nil, fmt.Errorf("reload updated review: %w", err)
	}

	uc.log.WithField("review_id", updated.ID).Info("master review updated")
	return updated, nil
}
