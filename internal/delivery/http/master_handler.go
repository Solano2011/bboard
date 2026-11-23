package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"bboard/internal/delivery/http/middleware"
	"bboard/internal/domain/master"
	masterUC "bboard/internal/usecase/master"

	"github.com/sirupsen/logrus"
)

type MasterHandler struct {
	createUC *masterUC.CreateProfileUseCase
	listUC   *masterUC.ListProfilesUseCase
	log      *logrus.Entry
}

func NewMasterHandler(
	createUC *masterUC.CreateProfileUseCase,
	listUC *masterUC.ListProfilesUseCase,
	log *logrus.Logger,
) *MasterHandler {
	return &MasterHandler{
		createUC: createUC,
		listUC:   listUC,
		log:      log.WithField("handler", "master"),
	}
}

type createMasterProfileRequest struct {
	Headline        string `json:"headline"`
	Bio             string `json:"bio"`
	Category        string `json:"category"`
	City            string `json:"city"`
	ExperienceYears int    `json:"experience_years"`
	MinPrice        int64  `json:"min_price"`
}

type masterProfileResponse struct {
	ID              int64     `json:"id"`
	UserID          int64     `json:"user_id"`
	Headline        string    `json:"headline"`
	Bio             string    `json:"bio"`
	Category        string    `json:"category"`
	City            string    `json:"city,omitempty"`
	ExperienceYears int       `json:"experience_years"`
	MinPrice        int64     `json:"min_price"`
	IsActive        bool      `json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Create обрабатывает POST /api/v1/masters (требует JWT).
func (h *MasterHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req createMasterProfileRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&req); err != nil {
		h.log.WithError(err).Warn("failed to decode create master profile request")
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON payload"})
		return
	}

	created, err := h.createUC.Execute(r.Context(), masterUC.CreateProfileInput{
		UserID:          userID,
		Headline:        req.Headline,
		Bio:             req.Bio,
		Category:        req.Category,
		City:            req.City,
		ExperienceYears: req.ExperienceYears,
		MinPrice:        req.MinPrice,
	})
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, mapMasterToResponse(created))
}

// List обрабатывает GET /api/v1/masters (публичный каталог мастеров).
func (h *MasterHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))

	profiles, err := h.listUC.Execute(r.Context(), masterUC.ListProfilesInput{
		Category: q.Get("category"),
		City:     q.Get("city"),
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	resp := make([]masterProfileResponse, 0, len(profiles))
	for _, p := range profiles {
		resp = append(resp, mapMasterToResponse(p))
	}

	writeJSON(w, http.StatusOK, resp)
}

func mapMasterToResponse(p *master.Profile) masterProfileResponse {
	return masterProfileResponse{
		ID:              p.ID,
		UserID:          p.UserID,
		Headline:        p.Headline,
		Bio:             p.Bio,
		Category:        p.Category,
		City:            p.City,
		ExperienceYears: p.ExperienceYears,
		MinPrice:        p.MinPrice,
		IsActive:        p.IsActive,
		CreatedAt:       p.CreatedAt,
		UpdatedAt:       p.UpdatedAt,
	}
}

func (h *MasterHandler) handleDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, master.ErrInvalidUserID),
		errors.Is(err, master.ErrEmptyHeadline),
		errors.Is(err, master.ErrHeadlineTooShort),
		errors.Is(err, master.ErrEmptyBio),
		errors.Is(err, master.ErrEmptyCategory),
		errors.Is(err, master.ErrInvalidExperience),
		errors.Is(err, master.ErrInvalidMinPrice):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})

	case errors.Is(err, master.ErrProfileAlreadyExists):
		writeJSON(w, http.StatusConflict, errorResponse{Error: err.Error()})

	case errors.Is(err, master.ErrProfileNotFound):
		writeJSON(w, http.StatusNotFound, errorResponse{Error: err.Error()})

	default:
		h.log.WithError(err).Error("unexpected internal error in master handler")
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
	}
}
