package user

import (
	"context"
	"time"
)

// Repository определяет контракт для работы с хранилищем пользователей.
type Repository interface {
	Create(ctx context.Context, u *User) error
	GetByID(ctx context.Context, id int64) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	List(ctx context.Context, search string, limit, offset int) ([]*User, error)
	Update(ctx context.Context, u *User) error
	UpdateRole(ctx context.Context, id int64, role Role, updatedAt time.Time) error
	PromoteToAdminByEmail(ctx context.Context, email string) error
}
