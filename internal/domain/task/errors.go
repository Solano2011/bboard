package task

import "errors"

var (
	// Ошибки бизнес-валидации
	ErrEmptyTitle       = errors.New("task title cannot be empty")
	ErrTitleTooShort    = errors.New("task title must be at least 5 characters long")
	ErrEmptyDescription = errors.New("task description cannot be empty")
	ErrEmptyCategory    = errors.New("task category cannot be empty")
	ErrInvalidBudget    = errors.New("task budget cannot be negative")
	ErrInvalidAuthor    = errors.New("task author id must be positive")
	ErrInvalidStatus    = errors.New("invalid task status transition")

	// Ошибки состояния и прав
	ErrTaskNotFound    = errors.New("task not found")
	ErrForbiddenAction = errors.New("user is not allowed to modify this task")

	ErrInvalidMaster     = errors.New("master id must be positive")
	ErrSelfAssignment    = errors.New("task author cannot assign themselves as master")
	ErrMasterNotAssigned = errors.New("cannot complete task without assigned master")
)
