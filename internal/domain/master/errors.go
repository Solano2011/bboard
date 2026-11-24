package master

import "errors"

var (
	// Ошибки бизнес-валидации
	ErrInvalidUserID     = errors.New("master user id must be positive")
	ErrEmptyHeadline     = errors.New("master headline cannot be empty")
	ErrHeadlineTooShort  = errors.New("master headline must be at least 5 characters long")
	ErrEmptyBio          = errors.New("master bio cannot be empty")
	ErrEmptyCategory     = errors.New("master category cannot be empty")
	ErrInvalidExperience = errors.New("experience years cannot be negative")
	ErrInvalidMinPrice   = errors.New("minimum price cannot be negative")
	ErrEmptyContactPhone = errors.New("master contact phone cannot be empty")
	ErrInvalidPhone      = errors.New("invalid contact phone format, expected +7XXXXXXXXXX")

	// Ошибки состояния и прав
	ErrProfileNotFound      = errors.New("master profile not found")
	ErrProfileAlreadyExists = errors.New("user already has a master profile")
	ErrForbiddenAction      = errors.New("user is not allowed to modify this master profile")
	ErrProfileInUse         = errors.New("master profile cannot be deleted because it is in use")
)
