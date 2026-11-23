package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"bboard/pkg/token"

	"github.com/sirupsen/logrus"
)

// Используем неэкспортируемый тип для ключа контекста, чтобы исключить коллизии
// с ключами других пакетов (стандартная идиома Go).
type contextKey string

const userIDContextKey contextKey = "user_id"

// TokenParser определяет минимальный контракт для проверки токена в middleware.
type TokenParser interface {
	ParseToken(tokenString string) (int64, error)
}

// Auth возвращает HTTP middleware для защиты приватных маршрутов.
func Auth(parser TokenParser, log *logrus.Logger) func(http.Handler) http.Handler {
	entry := log.WithField("middleware", "auth")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeUnauthorized(w, "missing authorization header")
				return
			}

			// Ожидаем формат: "Bearer <token>"
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
				writeUnauthorized(w, "invalid authorization header format")
				return
			}

			userID, err := parser.ParseToken(strings.TrimSpace(parts[1]))
			if err != nil {
				if errors.Is(err, token.ErrExpiredToken) {
					entry.WithError(err).Warn("request rejected: token expired")
					writeUnauthorized(w, "token has expired")
					return
				}

				entry.WithError(err).Warn("request rejected: invalid token")
				writeUnauthorized(w, "invalid token")
				return
			}

			// Кладём проверенный userID в контекст запроса и передаём управление дальше
			ctx := context.WithValue(r.Context(), userIDContextKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserIDFromContext извлекает ID авторизованного пользователя из контекста запроса.
// Возвращает (userID, true), если пользователь авторизован, или (0, false) в противном случае.
func UserIDFromContext(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value(userIDContextKey).(int64)
	return userID, ok && userID > 0
}

func writeUnauthorized(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": msg,
	})
}
