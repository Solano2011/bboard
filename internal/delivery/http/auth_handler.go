package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"bboard/internal/domain/user"
	"bboard/internal/usecase/auth"

	"github.com/sirupsen/logrus"
)

// AuthHandler обрабатывает HTTP-запросы, связанные с аутентификацией и регистрацией.
type AuthHandler struct {
	registerUC *auth.RegisterUseCase
	loginUC    *auth.LoginUseCase
	log        *logrus.Entry
}

// NewAuthHandler создаёт новый экземпляр HTTP-хэндлера авторизации.
func NewAuthHandler(
	registerUC *auth.RegisterUseCase,
	loginUC *auth.LoginUseCase,
	log *logrus.Logger,
) *AuthHandler {
	return &AuthHandler{
		registerUC: registerUC,
		loginUC:    loginUC,
		log:        log.WithField("handler", "auth"),
	}
}

// --- DTO (Data Transfer Objects) ---

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
	CreatedAt time.Time `json:"created_at"`
}

type loginResponse struct {
	AccessToken string       `json:"access_token"`
	User        userResponse `json:"user"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// --- HTTP Handlers ---

// Register обрабатывает POST /api/v1/auth/register.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req registerRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&req); err != nil {
		h.log.WithError(err).Warn("failed to decode register request body")
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Error: "invalid JSON payload",
		})
		return
	}

	input := auth.RegisterInput{
		Name:     req.Name,
		Email:    req.Email,
		Phone:    req.Phone,
		Password: req.Password,
	}

	createdUser, err := h.registerUC.Execute(r.Context(), input)
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, mapUserToResponse(createdUser))
}

// Login обрабатывает POST /api/v1/auth/login.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req loginRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&req); err != nil {
		h.log.WithError(err).Warn("failed to decode login request body")
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Error: "invalid JSON payload",
		})
		return
	}

	input := auth.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	}

	out, err := h.loginUC.Execute(r.Context(), input)
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	resp := loginResponse{
		AccessToken: out.Token,
		User:        mapUserToResponse(out.User),
	}

	writeJSON(w, http.StatusOK, resp)
}

// --- Helpers ---

func mapUserToResponse(u *user.User) userResponse {
	return userResponse{
		ID:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		Phone:     u.Phone,
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

	case errors.Is(err, user.ErrInvalidCreds):
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
