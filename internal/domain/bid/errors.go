package bid

import "errors"

var (
	ErrInvalidTaskID       = errors.New("task id must be positive")
	ErrInvalidMasterID     = errors.New("master profile id must be positive")
	ErrInvalidPrice        = errors.New("proposed price cannot be negative")
	ErrBidAlreadyExists    = errors.New("master has already submitted a response for this task")
	ErrSelfBidNotAllowed   = errors.New("task author cannot submit a response on their own task")
	ErrTaskNotOpenForBids  = errors.New("task is not open for new responses")
	ErrMasterProfileNeeded = errors.New("active master profile and at least one active service are required to submit a response")
	ErrForbiddenView       = errors.New("only task author can view responses for this task")
)
