package token

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid or malformed token")
	ErrExpiredToken = errors.New("token has expired")
)

// JWTManager отвечает за выпуск и валидацию JWT-токенов.
type JWTManager struct {
	secretKey []byte
	ttl       time.Duration
}

// NewJWTManager создаёт менеджер токенов с заданным секретным ключом и временем жизни (TTL).
func NewJWTManager(secretKey string, ttl time.Duration) (*JWTManager, error) {
	if len(secretKey) < 16 {
		return nil, errors.New("jwt secret key must be at least 16 characters long")
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}

	return &JWTManager{
		secretKey: []byte(secretKey),
		ttl:       ttl,
	}, nil
}

// GenerateToken выпускает подписанный HS256 JWT-токен для указанного userID.
// Сигнатура метода полностью совпадает с интерфейсом auth.TokenGenerator.
func (m *JWTManager) GenerateToken(userID int64) (string, error) {
	now := time.Now().UTC()

	claims := jwt.RegisteredClaims{
		Subject:   strconv.FormatInt(userID, 10),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secretKey)
	if err != nil {
		return "", fmt.Errorf("sign jwt token: %w", err)
	}

	return signed, nil
}

// ParseToken проверяет подпись и срок действия токена, возвращая userID из поля Subject.
func (m *JWTManager) ParseToken(tokenString string) (int64, error) {
	var claims jwt.RegisteredClaims

	token, err := jwt.ParseWithClaims(
		tokenString,
		&claims,
		func(t *jwt.Token) (any, error) {
			// Защита от подмены алгоритма (alg: none или RSA вместо HMAC)
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return m.secretKey, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return 0, ErrExpiredToken
		}
		return 0, ErrInvalidToken
	}

	if !token.Valid {
		return 0, ErrInvalidToken
	}

	userID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil || userID <= 0 {
		return 0, ErrInvalidToken
	}

	return userID, nil
}
