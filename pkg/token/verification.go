package token

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
)

// GenerateVerificationCode генерирует криптостойкий 6-значный цифровой код подтверждения.
func GenerateVerificationCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", fmt.Errorf("generate random verification code: %w", err)
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// HashVerificationCode возвращает SHA-256 hex-хэш кода подтверждения.
func HashVerificationCode(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}
