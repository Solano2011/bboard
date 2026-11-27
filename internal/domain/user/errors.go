package user

import "errors"

var (
	// Ошибки бизнес-валидации сущности
	ErrEmptyEmail    = errors.New("email cannot be empty")
	ErrInvalidEmail  = errors.New("invalid email format")
	ErrEmptyName     = errors.New("user name cannot be empty")
	ErrWeakPassword  = errors.New("password must be at least 8 characters long")
	ErrEmptyPassword = errors.New("password hash cannot be empty")
	ErrInvalidRole   = errors.New("invalid user role")

	// Ошибки состояния / хранилища / прав
	ErrUserNotFound        = errors.New("user not found")
	ErrUserAlreadyExist    = errors.New("user with this email already exists")
	ErrInvalidCreds        = errors.New("invalid email or password")
	ErrForbiddenRoleChange = errors.New("user is not allowed to change this role")
	ErrCannotChangeOwnRole = errors.New("admin cannot change their own role")
)
