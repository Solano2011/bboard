package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"bboard/internal/domain/user"
	"bboard/pkg/token"

	"github.com/sirupsen/logrus"
)

type contextKey string

const (
	userIDContextKey   contextKey = "user_id"
	userRoleContextKey contextKey = "user_role"
)

// TokenParser определяет минимальный контракт для проверки токена в middleware.
type TokenParser interface {
	ParseToken(tokenString string) (int64, error)
}

// UserProvider загружает актуального пользователя и его роль из БД.
type UserProvider interface {
	GetByID(ctx context.Context, id int64) (*user.User, error)
}

// Auth возвращает HTTP middleware для защиты приватных маршрутов с загрузкой актуальной роли из БД.
func Auth(parser TokenParser, users UserProvider, log *logrus.Logger) func(http.Handler) http.Handler {
	entry := log.WithField("middleware", "auth")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeUnauthorized(w, "missing authorization header")
				return
			}

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

			u, err := users.GetByID(r.Context(), userID)
			if err != nil {
				if errors.Is(err, user.ErrUserNotFound) {
					entry.WithField("user_id", userID).Warn("request rejected: user not found in db")
					writeUnauthorized(w, "user not found")
					return
				}
				entry.WithError(err).Error("failed to load user in auth middleware")
				writeInternalError(w)
				return
			}

			ctx := context.WithValue(r.Context(), userIDContextKey, u.ID)
			ctx = context.WithValue(ctx, userRoleContextKey, u.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalAuth извлекает userID и актуальную роль из БД, если передан валидный Bearer-токен.
func OptionalAuth(parser TokenParser, users UserProvider, log *logrus.Logger) func(http.Handler) http.Handler {
	entry := log.WithField("middleware", "optional_auth")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				next.ServeHTTP(w, r)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
				next.ServeHTTP(w, r)
				return
			}

			userID, err := parser.ParseToken(strings.TrimSpace(parts[1]))
			if err != nil {
				entry.WithError(err).Debug("optional auth: token invalid or expired, proceeding as guest")
				next.ServeHTTP(w, r)
				return
			}

			u, err := users.GetByID(r.Context(), userID)
			if err != nil {
				if errors.Is(err, user.ErrUserNotFound) {
					writeUnauthorized(w, "user not found")
					return
				}
				entry.WithError(err).Error("failed to load user in optional auth middleware")
				writeInternalError(w)
				return
			}

			ctx := context.WithValue(r.Context(), userIDContextKey, u.ID)
			ctx = context.WithValue(ctx, userRoleContextKey, u.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole ограничивает доступ к маршруту только указанными ролями (возвращает 403).
func RequireRole(roles ...user.Role) func(http.Handler) http.Handler {
	allowed := make(map[user.Role]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, ok := UserRoleFromContext(r.Context())
			if !ok {
				writeUnauthorized(w, "unauthorized")
				return
			}
			if _, exists := allowed[role]; !exists {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "forbidden: insufficient permissions"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// UserIDFromContext извлекает ID авторизованного пользователя из контекста запроса.
func UserIDFromContext(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value(userIDContextKey).(int64)
	return userID, ok && userID > 0
}

// UserRoleFromContext извлекает роль авторизованного пользователя из контекста запроса.
func UserRoleFromContext(ctx context.Context) (user.Role, bool) {
	role, ok := ctx.Value(userRoleContextKey).(user.Role)
	return role, ok && role.IsValid()
}

func writeUnauthorized(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": msg,
	})
}

func writeInternalError(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusInternalServerError)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": "internal server error",
	})
}
