package master

import "errors"

var (
	// Ошибки бизнес-валидации профиля и услуг
	ErrInvalidUserID     = errors.New("master user id must be positive")
	ErrInvalidProfileID  = errors.New("master profile id must be positive")
	ErrEmptyDisplayName  = errors.New("master display name cannot be empty")
	ErrDisplayNameTooLong = errors.New("master display name must not exceed 120 characters")
	ErrCityTooLong       = errors.New("city must not exceed 100 characters")
	ErrTelegramTooLong   = errors.New("telegram must not exceed 100 characters")
	ErrInvalidAvatarURL  = errors.New("avatar url must use http or https scheme")
	ErrEmptyTitle        = errors.New("service title cannot be empty")
	ErrTitleTooShort     = errors.New("service title must be at least 5 characters long")
	ErrTitleTooLong      = errors.New("service title must not exceed 200 characters")
	ErrEmptyCategory     = errors.New("service category cannot be empty")
	ErrInvalidCategory   = errors.New("invalid service category")
	ErrInvalidPrice      = errors.New("service price cannot be negative")
	ErrEmptyContactPhone = errors.New("master contact phone cannot be empty")
	ErrInvalidPhone      = errors.New("invalid contact phone format, expected +7XXXXXXXXXX")
	ErrDistrictTooLong   = errors.New("district must not exceed 100 characters")

	// Ошибки состояния, лимитов и прав
	ErrProfileNotFound          = errors.New("master profile not found")
	ErrProfileAlreadyExists     = errors.New("user already has a master profile")
	ErrServiceNotFound          = errors.New("master service not found")
	ErrServiceCategoryExists    = errors.New("active service in this category already exists")
	ErrMaxActiveServicesReached = errors.New("maximum 3 active services allowed per master profile")
	ErrForbiddenAction          = errors.New("user is not allowed to modify this master profile or service")
	ErrProfileInUse             = errors.New("master profile cannot be deleted because it is in use")
)
