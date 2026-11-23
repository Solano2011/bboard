package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// responseWriterWrapper перехватывает код статуса HTTP-ответа для логирования.
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriterWrapper) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Logger возвращает HTTP middleware для структурированного логирования всех входящих запросов.
func Logger(log *logrus.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			reqID := generateRequestID()

			// Возвращаем X-Request-ID в заголовке ответа для удобной отладки
			w.Header().Set("X-Request-ID", reqID)

			wrapped := &responseWriterWrapper{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			next.ServeHTTP(wrapped, r)

			duration := time.Since(start)
			entry := log.WithFields(logrus.Fields{
				"request_id":  reqID,
				"method":      r.Method,
				"path":        r.URL.Path,
				"status":      wrapped.statusCode,
				"duration_ms": duration.Milliseconds(),
				"remote_addr": r.RemoteAddr,
			})

			switch {
			case wrapped.statusCode >= 500:
				entry.Error("http request failed")
			case wrapped.statusCode >= 400:
				entry.Warn("http request client error")
			default:
				entry.Info("http request completed")
			}
		})
	}
}

func generateRequestID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(b)
}
