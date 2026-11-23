package user

import (
	"strings"
	"time"
)

// User представляет зарегистрированного пользователя платформы.
// Один аккаунт может выступать и как заказчик (создавать задачи),
// и как исполнитель (если у него заполнена анкета мастера).
type User struct {
	ID           int64
	Name         string
	Email        string
	Phone        string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// NewUser создаёт и валидирует новую доменную сущность пользователя перед сохранением.
// На вход ожидается уже захэшированный пароль (passwordHash).
func NewUser(name, email, phone, passwordHash string) (*User, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrEmptyName
	}

	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return nil, ErrEmptyEmail
	}
	if !strings.Contains(email, "@") || !strings.Contains(email, ".") {
		return nil, ErrInvalidEmail
	}

	if strings.TrimSpace(passwordHash) == "" {
		return nil, ErrEmptyPassword
	}

	now := time.Now().UTC()

	return &User{
		Name:         name,
		Email:        email,
		Phone:        strings.TrimSpace(phone),
		PasswordHash: passwordHash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

// UpdateProfile обновляет контактные данные пользователя.
func (u *User) UpdateProfile(name, phone string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrEmptyName
	}

	u.Name = name
	u.Phone = strings.TrimSpace(phone)
	u.UpdatedAt = time.Now().UTC()
	return nil
}

// ChangePassword устанавливает новый хэш пароля.
func (u *User) ChangePassword(newPasswordHash string) error {
	if strings.TrimSpace(newPasswordHash) == "" {
		return ErrEmptyPassword
	}

	u.PasswordHash = newPasswordHash
	u.UpdatedAt = time.Now().UTC()
	return nil
}
