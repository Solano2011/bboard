package user

import "context"

// Repository определяет контракт для работы с хранилищем пользователей.
type Repository interface {
	// Create сохраняет нового пользователя и заполняет его сгенерированный ID.
	Create(ctx context.Context, u *User) error

	// GetByID возвращает пользователя по уникальному идентификатору.
	// Возвращает ErrUserNotFound, если запись не найдена.
	GetByID(ctx context.Context, id int64) (*User, error)

	// GetByEmail возвращает пользователя по email (регистронезависимо).
	// Возвращает ErrUserNotFound, если запись не найдена.
	GetByEmail(ctx context.Context, email string) (*User, error)

	// Update сохраняет изменения существующего пользователя.
	Update(ctx context.Context, u *User) error
}
