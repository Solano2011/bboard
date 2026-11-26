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
	reviewHandler *ReviewHandler,
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
	mux.HandleFunc("POST /api/v1/auth/refresh", authHandler.Refresh)
	mux.HandleFunc("POST /api/v1/auth/logout", authHandler.Logout)

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

	// Объявления заказчиков
	mux.Handle("GET /api/v1/tasks", optAuthMW(http.HandlerFunc(taskHandler.List)))
	mux.Handle("POST /api/v1/tasks", authMW(http.HandlerFunc(taskHandler.Create)))
	mux.Handle("PUT /api/v1/tasks/{id}", authMW(http.HandlerFunc(taskHandler.Update)))
	mux.Handle("POST /api/v1/tasks/{id}/assign", authMW(http.HandlerFunc(taskHandler.AssignMaster)))
	mux.Handle("POST /api/v1/tasks/{id}/mark-completed", authMW(http.HandlerFunc(taskHandler.MarkCompleted)))
	mux.Handle("POST /api/v1/tasks/{id}/complete", authMW(http.HandlerFunc(taskHandler.Complete)))
	mux.Handle("POST /api/v1/tasks/{id}/cancel", authMW(http.HandlerFunc(taskHandler.Cancel)))

	// Отклики мастеров на объявления
	mux.Handle("POST /api/v1/tasks/{id}/bids", authMW(http.HandlerFunc(bidHandler.Create)))
	mux.Handle("GET /api/v1/tasks/{id}/bids", authMW(http.HandlerFunc(bidHandler.ListByTask)))

	// Отзывы по завершённым заказам
	mux.Handle("POST /api/v1/tasks/{id}/review", authMW(http.HandlerFunc(reviewHandler.Create)))
	mux.Handle("PUT /api/v1/reviews/{id}", authMW(http.HandlerFunc(reviewHandler.Update)))
	mux.Handle("DELETE /api/v1/reviews/{id}", authMW(http.HandlerFunc(reviewHandler.Delete)))

	// Профили мастеров и анкеты услуг (с optAuthMW для фильтрации скрытых контактов)
	mux.Handle("GET /api/v1/master-profile/me", authMW(http.HandlerFunc(masterHandler.GetMyProfile)))
	mux.Handle("PUT /api/v1/master-profile/me", authMW(http.HandlerFunc(masterHandler.UpsertMyProfile)))
	mux.Handle("GET /api/v1/masters", optAuthMW(http.HandlerFunc(masterHandler.ListServices)))
	mux.Handle("GET /api/v1/masters/{id}", optAuthMW(http.HandlerFunc(masterHandler.GetPublicProfile)))
	mux.Handle("GET /api/v1/master-services", optAuthMW(http.HandlerFunc(masterHandler.ListServices)))
	mux.Handle("POST /api/v1/master-services", authMW(http.HandlerFunc(masterHandler.CreateService)))
	mux.Handle("PUT /api/v1/master-services/{id}", authMW(http.HandlerFunc(masterHandler.UpdateService)))
	mux.Handle("DELETE /api/v1/master-services/{id}", authMW(http.HandlerFunc(masterHandler.DeleteService)))

	fileServer := http.FileServer(http.Dir("web"))
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		fileServer.ServeHTTP(w, r)
	}))
	// Оборачиваем весь роутер в глобальный middleware логирования
	return middleware.Logger(log)(mux)
}
