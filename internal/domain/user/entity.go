package user

import (
	"net/mail"
	"strings"
	"time"
	"unicode"
)

type Role string

const (
	RoleUser      Role = "user"
	RoleModerator Role = "moderator"
	RoleAdmin     Role = "admin"
)

// IsValid проверяет, является ли роль допустимой в системе.
func (r Role) IsValid() bool {
	return r == RoleUser || r == RoleModerator || r == RoleAdmin
}

// IsModeratorOrAdmin возвращает true для модератора и администратора.
func (r Role) IsModeratorOrAdmin() bool {
	return r == RoleModerator || r == RoleAdmin
}

// IsAdmin возвращает true только для администратора.
func (r Role) IsAdmin() bool {
	return r == RoleAdmin
}

// User представляет зарегистрированного пользователя платформы.
type User struct {
	ID              int64
	Name            string
	Email           string
	Phone           string
	Role            Role
	PasswordHash    string
	EmailVerifiedAt *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// NormalizeAndValidateEmail нормализует email и выполняет строгую проверку формата (без display-name, CR/LF и пробелов).
func NormalizeAndValidateEmail(raw string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(raw))
	if email == "" {
		return "", ErrEmptyEmail
	}
	if len(email) > 255 {
		return "", ErrInvalidEmail
	}
	for _, r := range email {
		if r == '\r' || r == '\n' || unicode.IsSpace(r) {
			return "", ErrInvalidEmail
		}
	}

	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Name != "" || addr.Address != email {
		return "", ErrInvalidEmail
	}

	atIdx := strings.LastIndex(email, "@")
	if atIdx <= 0 || atIdx >= len(email)-1 {
		return "", ErrInvalidEmail
	}
	domain := email[atIdx+1:]
	if !strings.Contains(domain, ".") || strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") {
		return "", ErrInvalidEmail
	}

	return email, nil
}

// NewUser создаёт и валидирует новую доменную сущность пользователя перед сохранением.
func NewUser(name, email, phone, passwordHash string) (*User, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrEmptyName
	}

	normalizedEmail, err := NormalizeAndValidateEmail(email)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(passwordHash) == "" {
		return nil, ErrEmptyPassword
	}

	now := time.Now().UTC()

	return &User{
		Name:         name,
		Email:        normalizedEmail,
		Phone:        strings.TrimSpace(phone),
		Role:         RoleUser,
		PasswordHash: passwordHash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

// ChangeModeratorRole изменяет роль пользователя между user и moderator (доступно только admin).
func (u *User) ChangeModeratorRole(actorID int64, actorRole Role, newRole Role) error {
	if !actorRole.IsAdmin() {
		return ErrForbiddenRoleChange
	}
	if u.ID == actorID {
		return ErrCannotChangeOwnRole
	}
	if u.Role == RoleAdmin {
		return ErrForbiddenRoleChange
	}
	if !newRole.IsValid() {
		return ErrInvalidRole
	}
	if newRole == RoleAdmin {
		return ErrForbiddenRoleChange
	}

	u.Role = newRole
	u.UpdatedAt = time.Now().UTC()
	return nil
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
