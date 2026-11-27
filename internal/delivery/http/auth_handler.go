package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"bboard/internal/delivery/http/middleware"
	domainAuth "bboard/internal/domain/auth"
	"bboard/internal/domain/user"
	"bboard/internal/usecase/auth"

	"github.com/sirupsen/logrus"
)

const (
	refreshTokenCookieName = "refresh_token"
	refreshTokenCookiePath = "/api/v1/auth"
)

// AuthHandler обрабатывает HTTP-запросы, связанные с аутентификацией и регистрацией.
type AuthHandler struct {
	registerUC *auth.RegisterUseCase
	loginUC    *auth.LoginUseCase
	refreshUC  *auth.RefreshUseCase
	logoutUC   *auth.LogoutUseCase
	userRepo   user.Repository
	refreshTTL time.Duration
	isProd     bool
	log        *logrus.Entry
}

// NewAuthHandler создаёт новый экземпляр HTTP-хэндлера авторизации.
func NewAuthHandler(
	registerUC *auth.RegisterUseCase,
	loginUC *auth.LoginUseCase,
	refreshUC *auth.RefreshUseCase,
	logoutUC *auth.LogoutUseCase,
	userRepo user.Repository,
	refreshTTL time.Duration,
	isProd bool,
	log *logrus.Logger,
) *AuthHandler {
	return &AuthHandler{
		registerUC: registerUC,
		loginUC:    loginUC,
		refreshUC:  refreshUC,
		logoutUC:   logoutUC,
		userRepo:   userRepo,
		refreshTTL: refreshTTL,
		isProd:     isProd,
		log:        log.WithField("handler", "auth"),
	}
}

type registerRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type userResponse struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone,omitempty"`
	Role      user.Role `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type authResponse struct {
	AccessToken string       `json:"access_token"`
	User        userResponse `json:"user"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// Register обрабатывает POST /api/v1/auth/register.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req registerRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&req); err != nil {
		h.log.WithError(err).Warn("failed to decode register request body")
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON payload"})
		return
	}

	out, err := h.registerUC.Execute(r.Context(), auth.RegisterInput{
		Name:     req.Name,
		Email:    req.Email,
		Phone:    req.Phone,
		Password: req.Password,
	})
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	h.setRefreshCookie(w, out.RefreshToken)
	writeJSON(w, http.StatusCreated, authResponse{
		AccessToken: out.AccessToken,
		User:        mapUserToResponse(out.User),
	})
}

// Login обрабатывает POST /api/v1/auth/login.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req loginRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&req); err != nil {
		h.log.WithError(err).Warn("failed to decode login request body")
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON payload"})
		return
	}

	out, err := h.loginUC.Execute(r.Context(), auth.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	h.setRefreshCookie(w, out.RefreshToken)
	writeJSON(w, http.StatusOK, authResponse{
		AccessToken: out.AccessToken,
		User:        mapUserToResponse(out.User),
	})
}

// Refresh обрабатывает POST /api/v1/auth/refresh.
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshTokenCookieName)
	if err != nil || cookie.Value == "" {
		h.clearRefreshCookie(w)
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "missing refresh token"})
		return
	}

	out, err := h.refreshUC.Execute(r.Context(), cookie.Value)
	if err != nil {
		h.clearRefreshCookie(w)
		h.handleDomainError(w, err)
		return
	}

	h.setRefreshCookie(w, out.RefreshToken)
	writeJSON(w, http.StatusOK, authResponse{
		AccessToken: out.AccessToken,
		User:        mapUserToResponse(out.User),
	})
}

// Logout обрабатывает POST /api/v1/auth/logout.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(refreshTokenCookieName); err == nil && cookie.Value != "" {
		if err := h.logoutUC.Execute(r.Context(), cookie.Value); err != nil {
			h.log.WithError(err).Error("failed to revoke refresh token on logout")
		}
	}

	h.clearRefreshCookie(w)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Me обрабатывает GET /api/v1/auth/me и возвращает актуальные данные пользователя из БД.
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}

	u, err := h.userRepo.GetByID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "user not found"})
			return
		}
		h.log.WithError(err).Error("failed to load current user in /auth/me")
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
		return
	}

	writeJSON(w, http.StatusOK, mapUserToResponse(u))
}

func (h *AuthHandler) setRefreshCookie(w http.ResponseWriter, rawToken string) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshTokenCookieName,
		Value:    rawToken,
		Path:     refreshTokenCookiePath,
		MaxAge:   int(h.refreshTTL.Seconds()),
		HttpOnly: true,
		Secure:   h.isProd,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *AuthHandler) clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshTokenCookieName,
		Value:    "",
		Path:     refreshTokenCookiePath,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.isProd,
		SameSite: http.SameSiteLaxMode,
	})
}

func mapUserToResponse(u *user.User) userResponse {
	role := u.Role
	if role == "" {
		role = user.RoleUser
	}
	return userResponse{
		ID:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		Phone:     u.Phone,
		Role:      role,
		CreatedAt: u.CreatedAt,
	}
}

func (h *AuthHandler) handleDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, user.ErrEmptyName),
		errors.Is(err, user.ErrEmptyEmail),
		errors.Is(err, user.ErrInvalidEmail),
		errors.Is(err, user.ErrWeakPassword):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})

	case errors.Is(err, user.ErrUserAlreadyExist):
		writeJSON(w, http.StatusConflict, errorResponse{Error: err.Error()})

	case errors.Is(err, user.ErrInvalidCreds),
		errors.Is(err, domainAuth.ErrInvalidRefreshToken),
		errors.Is(err, domainAuth.ErrRefreshTokenExpired),
		errors.Is(err, domainAuth.ErrRefreshTokenRevoked):
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: err.Error()})

	default:
		h.log.WithError(err).Error("unexpected internal error in auth handler")
		writeJSON(w, http.StatusInternalServerError, errorResponse{
			Error: "internal server error",
		})
	}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
