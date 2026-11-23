package hash

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword принимает сырой пароль и возвращает его bcrypt-хэш.
// Параметр cost определяет вычислительную сложность (если передано 0, используется bcrypt.DefaultCost = 10).
func HashPassword(password string, cost int) (string, error) {
	if cost == 0 {
		cost = bcrypt.DefaultCost
	}

	bytes, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", fmt.Errorf("failed to generate bcrypt hash: %w", err)
	}

	return string(bytes), nil
}

// CheckPassword сверяет сырой пароль с сохранённым bcrypt-хэшем.
// Возвращает true, если пароль верный, и false — если неверный.
func CheckPassword(password, hashedPassword string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return false, nil
		}
		return false, fmt.Errorf("failed to compare bcrypt hash: %w", err)
	}

	return true, nil
}
