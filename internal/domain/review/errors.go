package review

import "errors"

var (
	ErrInvalidTaskID       = errors.New("task id must be positive")
	ErrInvalidMasterID     = errors.New("master profile id must be positive")
	ErrInvalidAuthorID     = errors.New("author user id must be positive")
	ErrReviewNotFound      = errors.New("review not found")
	ErrReviewAlreadyExists = errors.New("review already exists for this task")
	ErrInvalidRating       = errors.New("rating must be between 1 and 5")
	ErrTaskNotCompleted    = errors.New("review can only be left for completed tasks")
	ErrForbiddenReview     = errors.New("user is not allowed to create or modify this review")
)
