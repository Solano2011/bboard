package task

import "errors"

var (
	// Ошибки бизнес-валидации
	ErrEmptyTitle       = errors.New("task title cannot be empty")
	ErrTitleTooShort    = errors.New("task title must be at least 5 characters long")
	ErrEmptyDescription = errors.New("task description cannot be empty")
	ErrEmptyCategory    = errors.New("task category cannot be empty")
	ErrInvalidCategory  = errors.New("invalid task category")
	ErrInvalidBudget    = errors.New("task budget cannot be negative")
	ErrInvalidAuthor    = errors.New("task author id must be positive")
	ErrInvalidStatus    = errors.New("invalid task status transition")
	ErrDistrictTooLong  = errors.New("district must not exceed 100 characters")
	ErrInvalidPhone     = errors.New("invalid contact phone format, expected +7XXXXXXXXXX")
	ErrAddressTooLong   = errors.New("work address must not exceed 255 characters")

	// Ошибки состояния и прав
	ErrTaskNotFound    = errors.New("task not found")
	ErrForbiddenAction = errors.New("user is not allowed to modify this task")

	ErrInvalidMaster     = errors.New("master id must be positive")
	ErrSelfAssignment    = errors.New("task author cannot assign themselves as master")
	ErrMasterNotAssigned = errors.New("cannot complete task without assigned master")

	// Ошибки работы с фотографиями
	ErrPhotoNotFound        = errors.New("task photo not found")
	ErrTooManyPhotos        = errors.New("maximum 3 photos allowed per task")
	ErrPhotoTooLarge        = errors.New("photo size exceeds 5 MB limit")
	ErrEmptyPhotoFile       = errors.New("photo file cannot be empty")
	ErrUnsupportedPhotoType = errors.New("unsupported photo format, allowed: jpg, png, webp")
)
