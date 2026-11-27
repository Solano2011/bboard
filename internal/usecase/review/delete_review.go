package review

import (
	"context"

	"bboard/internal/domain/review"
	"bboard/internal/domain/user"

	"github.com/sirupsen/logrus"
)

type DeleteReviewInput struct {
	ReviewID    int64
	ActorUserID int64
	ActorRole   user.Role
}

type DeleteReviewUseCase struct {
	reviewRepo review.Repository
	log        *logrus.Entry
}

func NewDeleteReviewUseCase(reviewRepo review.Repository, log *logrus.Logger) *DeleteReviewUseCase {
	return &DeleteReviewUseCase{
		reviewRepo: reviewRepo,
		log:        log.WithField("usecase", "review.delete"),
	}
}

func (uc *DeleteReviewUseCase) Execute(ctx context.Context, input DeleteReviewInput) error {
	if input.ReviewID <= 0 {
		return review.ErrReviewNotFound
	}

	rev, err := uc.reviewRepo.GetByID(ctx, input.ReviewID)
	if err != nil {
		return err
	}

	if rev.AuthorUserID != input.ActorUserID && !input.ActorRole.IsModeratorOrAdmin() {
		return review.ErrForbiddenReview
	}

	if err := uc.reviewRepo.Delete(ctx, rev.ID); err != nil {
		return err
	}

	uc.log.WithFields(logrus.Fields{
		"review_id":  rev.ID,
		"actor_id":   input.ActorUserID,
		"actor_role": input.ActorRole,
	}).Info("master review deleted")
	return nil
}
