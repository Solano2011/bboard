package http

import (
	"net/http"

	"bboard/internal/delivery/http/middleware"

	"github.com/sirupsen/logrus"
)

// NewRouter регистрирует все HTTP-маршруты приложения и оборачивает их в middleware.
func NewRouter(
	authHandler *AuthHandler,
	taskHandler *TaskHandler,
	masterHandler *MasterHandler,
	bidHandler *BidHandler,
	tokenParser middleware.TokenParser,
	log *logrus.Logger,
) http.Handler {
	mux := http.NewServeMux()
	authMW := middleware.Auth(tokenParser, log)
	optAuthMW := middleware.OptionalAuth(tokenParser, log)

	// Проверка работоспособности сервера (Healthcheck)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Публичные маршруты аутентификации
	mux.HandleFunc("POST /api/v1/auth/register", authHandler.Register)
	mux.HandleFunc("POST /api/v1/auth/login", authHandler.Login)

	// Защищённый маршрут текущего пользователя
	meHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.UserIDFromContext(r.Context())
		if !ok {
			writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]int64{"user_id": userID})
	})
	mux.Handle("GET /api/v1/auth/me", authMW(meHandler))

	// Публичный просмотр объявлений (с опциональной авторизацией) и защищённые действия с задачами
	mux.Handle("GET /api/v1/tasks", optAuthMW(http.HandlerFunc(taskHandler.List)))
	mux.Handle("POST /api/v1/tasks", authMW(http.HandlerFunc(taskHandler.Create)))
	mux.Handle("POST /api/v1/tasks/{id}/assign", authMW(http.HandlerFunc(taskHandler.AssignMaster)))
	mux.Handle("POST /api/v1/tasks/{id}/complete", authMW(http.HandlerFunc(taskHandler.Complete)))
	mux.Handle("POST /api/v1/tasks/{id}/cancel", authMW(http.HandlerFunc(taskHandler.Cancel)))

	// Отклики мастеров на объявления
	mux.Handle("POST /api/v1/tasks/{id}/bids", authMW(http.HandlerFunc(bidHandler.Create)))
	mux.Handle("GET /api/v1/tasks/{id}/bids", authMW(http.HandlerFunc(bidHandler.ListByTask)))

	// Публичный каталог мастеров, создание и удаление анкеты
	mux.HandleFunc("GET /api/v1/masters", masterHandler.List)
	mux.HandleFunc("GET /api/v1/masters/{id}", masterHandler.GetByID)
	mux.Handle("POST /api/v1/masters", authMW(http.HandlerFunc(masterHandler.Create)))
	mux.Handle("DELETE /api/v1/masters/{id}", authMW(http.HandlerFunc(masterHandler.Delete)))

	mux.Handle("/", http.FileServer(http.Dir("web")))
	// Оборачиваем весь роутер в глобальный middleware логирования
	return middleware.Logger(log)(mux)
}
