package bid

import "errors"

var (
	ErrInvalidTaskID       = errors.New("task id must be positive")
	ErrInvalidMasterID     = errors.New("master id must be positive")
	ErrInvalidPrice        = errors.New("proposed price cannot be negative")
	ErrEmptyComment        = errors.New("bid comment cannot be empty")
	ErrCommentTooShort     = errors.New("bid comment must be at least 5 characters long")
	ErrBidAlreadyExists    = errors.New("master has already submitted a bid for this task")
	ErrSelfBidNotAllowed   = errors.New("task author cannot submit a bid on their own task")
	ErrTaskNotOpenForBids  = errors.New("task is not open for new bids")
	ErrMasterProfileNeeded = errors.New("active master profile is required to submit a bid")
	ErrForbiddenView       = errors.New("only task author can view bids for this task")
)
