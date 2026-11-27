package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"bboard/internal/delivery/http/middleware"
	"bboard/internal/domain/master"
	"bboard/internal/domain/review"
	masterUC "bboard/internal/usecase/master"

	"github.com/sirupsen/logrus"
)

type MasterHandler struct {
	upsertProfileUC    *masterUC.UpsertProfileUseCase
	getMyProfileUC     *masterUC.GetMyProfileUseCase
	getPublicProfileUC *masterUC.GetPublicProfileUseCase
	deleteMyProfileUC  *masterUC.DeleteMyProfileUseCase
	createServiceUC    *masterUC.CreateServiceUseCase
	updateServiceUC    *masterUC.UpdateServiceUseCase
	deleteServiceUC    *masterUC.DeleteServiceUseCase
	listServicesUC     *masterUC.ListServicesUseCase
	log                *logrus.Entry
}

func NewMasterHandler(
	upsertProfileUC *masterUC.UpsertProfileUseCase,
	getMyProfileUC *masterUC.GetMyProfileUseCase,
	getPublicProfileUC *masterUC.GetPublicProfileUseCase,
	deleteMyProfileUC *masterUC.DeleteMyProfileUseCase,
	createServiceUC *masterUC.CreateServiceUseCase,
	updateServiceUC *masterUC.UpdateServiceUseCase,
	deleteServiceUC *masterUC.DeleteServiceUseCase,
	listServicesUC *masterUC.ListServicesUseCase,
	log *logrus.Logger,
) *MasterHandler {
	return &MasterHandler{
		upsertProfileUC:    upsertProfileUC,
		getMyProfileUC:     getMyProfileUC,
		getPublicProfileUC: getPublicProfileUC,
		deleteMyProfileUC:  deleteMyProfileUC,
		createServiceUC:    createServiceUC,
		updateServiceUC:    updateServiceUC,
		deleteServiceUC:    deleteServiceUC,
		listServicesUC:     listServicesUC,
		log:                log.WithField("handler", "master"),
	}
}

type upsertProfileRequest struct {
	DisplayName      string `json:"display_name"`
	AvatarURL        string `json:"avatar_url"`
	About            string `json:"about"`
	City             string `json:"city"`
	District         string `json:"district"`
	ContactPhone     string `json:"contact_phone"`
	Telegram         string `json:"telegram"`
	IsContactsPublic *bool  `json:"is_contacts_public"`
}

type serviceRequest struct {
	Category              string `json:"category"`
	Title                 string `json:"title"`
	ExperienceDescription string `json:"experience_description"`
	Price                 int64  `json:"price"`
	IsActive              *bool  `json:"is_active"`
}

type masterProfileResponse struct {
	ID                   int64     `json:"id"`
	UserID               int64     `json:"user_id"`
	DisplayName          string    `json:"display_name"`
	AvatarURL            string    `json:"avatar_url,omitempty"`
	About                string    `json:"about"`
	City                 string    `json:"city,omitempty"`
	District             string    `json:"district,omitempty"`
	ContactPhone         string    `json:"contact_phone,omitempty"`
	Telegram             string    `json:"telegram,omitempty"`
	IsContactsPublic     bool      `json:"is_contacts_public"`
	RatingAvg            float64   `json:"rating_avg"`
	ReviewsCount         int       `json:"reviews_count"`
	CompletedOrdersCount int       `json:"completed_orders_count"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type masterServiceResponse struct {
	ID                    int64     `json:"id"`
	MasterProfileID       int64     `json:"master_profile_id"`
	Category              string    `json:"category"`
	Title                 string    `json:"title"`
	ExperienceDescription string    `json:"experience_description"`
	Price                 int64     `json:"price"`
	IsActive              bool      `json:"is_active"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

type catalogServiceResponse struct {
	masterServiceResponse
	MasterDisplayName    string  `json:"master_display_name"`
	MasterAvatarURL      string  `json:"master_avatar_url,omitempty"`
	MasterCity           string  `json:"master_city,omitempty"`
	MasterDistrict       string  `json:"master_district,omitempty"`
	RatingAvg            float64 `json:"rating_avg"`
	ReviewsCount         int     `json:"reviews_count"`
	CompletedOrdersCount int     `json:"completed_orders_count"`
}

type myProfileResponse struct {
	Profile  masterProfileResponse   `json:"profile"`
	Services []masterServiceResponse `json:"services"`
}

type publicProfileResponse struct {
	Profile         masterProfileResponse   `json:"profile"`
	Services        []masterServiceResponse `json:"services"`
	Reviews         []reviewResponse        `json:"reviews"`
	ContactsVisible bool                    `json:"contacts_visible"`
}

// GetMyProfile обрабатывает GET /api/v1/master-profile/me.
func (h *MasterHandler) GetMyProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}

	out, err := h.getMyProfileUC.Execute(r.Context(), userID)
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	services := make([]masterServiceResponse, 0, len(out.Services))
	for _, s := range out.Services {
		services = append(services, mapServiceToResponse(s))
	}

	writeJSON(w, http.StatusOK, myProfileResponse{
		Profile:  mapProfileToResponse(out.Profile),
		Services: services,
	})
}

// UpsertMyProfile обрабатывает PUT /api/v1/master-profile/me.
func (h *MasterHandler) UpsertMyProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req upsertProfileRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON payload"})
		return
	}

	profile, err := h.upsertProfileUC.Execute(r.Context(), masterUC.UpsertProfileInput{
		UserID:           userID,
		DisplayName:      req.DisplayName,
		AvatarURL:        req.AvatarURL,
		About:            req.About,
		City:             req.City,
		District:         req.District,
		ContactPhone:     req.ContactPhone,
		Telegram:         req.Telegram,
		IsContactsPublic: req.IsContactsPublic,
	})
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, mapProfileToResponse(profile))
}

// DeleteMyProfile обрабатывает DELETE /api/v1/master-profile/me.
func (h *MasterHandler) DeleteMyProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}

	if err := h.deleteMyProfileUC.Execute(r.Context(), userID); err != nil {
		h.handleDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// GetPublicProfile обрабатывает GET /api/v1/masters/{id}.
func (h *MasterHandler) GetPublicProfile(w http.ResponseWriter, r *http.Request) {
	profileID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || profileID <= 0 {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid master profile id in path"})
		return
	}

	viewerUserID, _ := middleware.UserIDFromContext(r.Context())
	viewerRole, _ := middleware.UserRoleFromContext(r.Context())
	out, err := h.getPublicProfileUC.Execute(r.Context(), profileID, viewerUserID, viewerRole)
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	services := make([]masterServiceResponse, 0, len(out.Services))
	for _, s := range out.Services {
		services = append(services, mapServiceToResponse(s))
	}

	reviews := make([]reviewResponse, 0, len(out.Reviews))
	for _, rev := range out.Reviews {
		reviews = append(reviews, mapReviewToResponse(rev))
	}

	writeJSON(w, http.StatusOK, publicProfileResponse{
		Profile:         mapProfileToResponse(out.Profile),
		Services:        services,
		Reviews:         reviews,
		ContactsVisible: out.ContactsVisible,
	})
}

// ListServices обрабатывает GET /api/v1/master-services (и alias GET /api/v1/masters).
func (h *MasterHandler) ListServices(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var limit, offset int
	if raw := q.Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid limit query parameter"})
			return
		}
		limit = parsed
	}
	if raw := q.Get("offset"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid offset query parameter"})
			return
		}
		offset = parsed
	}

	items, err := h.listServicesUC.Execute(r.Context(), masterUC.ListServicesInput{
		Category: q.Get("category"),
		City:     q.Get("city"),
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	resp := make([]catalogServiceResponse, 0, len(items))
	for _, item := range items {
		resp = append(resp, catalogServiceResponse{
			masterServiceResponse: mapServiceToResponse(&item.Service),
			MasterDisplayName:     item.MasterDisplayName,
			MasterAvatarURL:       item.MasterAvatarURL,
			MasterCity:            item.MasterCity,
			MasterDistrict:        item.MasterDistrict,
			RatingAvg:             item.RatingAvg,
			ReviewsCount:          item.ReviewsCount,
			CompletedOrdersCount:  item.CompletedOrdersCount,
		})
	}

	writeJSON(w, http.StatusOK, resp)
}

// CreateService обрабатывает POST /api/v1/master-services.
func (h *MasterHandler) CreateService(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req serviceRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON payload"})
		return
	}

	svc, err := h.createServiceUC.Execute(r.Context(), masterUC.CreateServiceInput{
		ActorUserID:           userID,
		Category:              req.Category,
		Title:                 req.Title,
		ExperienceDescription: req.ExperienceDescription,
		Price:                 req.Price,
	})
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, mapServiceToResponse(svc))
}

// UpdateService обрабатывает PUT /api/v1/master-services/{id}.
func (h *MasterHandler) UpdateService(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}
	actorRole, _ := middleware.UserRoleFromContext(r.Context())

	serviceID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || serviceID <= 0 {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid service id in path"})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req serviceRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON payload"})
		return
	}

	svc, err := h.updateServiceUC.Execute(r.Context(), masterUC.UpdateServiceInput{
		ServiceID:             serviceID,
		ActorUserID:           userID,
		ActorRole:             actorRole,
		Category:              req.Category,
		Title:                 req.Title,
		ExperienceDescription: req.ExperienceDescription,
		Price:                 req.Price,
		IsActive:              req.IsActive,
	})
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, mapServiceToResponse(svc))
}

// DeleteService обрабатывает DELETE /api/v1/master-services/{id}.
func (h *MasterHandler) DeleteService(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}
	actorRole, _ := middleware.UserRoleFromContext(r.Context())

	serviceID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || serviceID <= 0 {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid service id in path"})
		return
	}

	if err := h.deleteServiceUC.Execute(r.Context(), masterUC.DeleteServiceInput{
		ServiceID:   serviceID,
		ActorUserID: userID,
		ActorRole:   actorRole,
	}); err != nil {
		h.handleDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func mapProfileToResponse(p *master.Profile) masterProfileResponse {
	return masterProfileResponse{
		ID:                   p.ID,
		UserID:               p.UserID,
		DisplayName:          p.DisplayName,
		AvatarURL:            p.AvatarURL,
		About:                p.About,
		City:                 p.City,
		District:             p.District,
		ContactPhone:         p.ContactPhone,
		Telegram:             p.Telegram,
		IsContactsPublic:     p.IsContactsPublic,
		RatingAvg:            p.RatingAvg,
		ReviewsCount:         p.ReviewsCount,
		CompletedOrdersCount: p.CompletedOrdersCount,
		CreatedAt:            p.CreatedAt,
		UpdatedAt:            p.UpdatedAt,
	}
}

func mapServiceToResponse(s *master.Service) masterServiceResponse {
	return masterServiceResponse{
		ID:                    s.ID,
		MasterProfileID:       s.MasterProfileID,
		Category:              s.Category,
		Title:                 s.Title,
		ExperienceDescription: s.ExperienceDescription,
		Price:                 s.Price,
		IsActive:              s.IsActive,
		CreatedAt:             s.CreatedAt,
		UpdatedAt:             s.UpdatedAt,
	}
}

func mapReviewToResponse(rev *review.Review) reviewResponse {
	return reviewResponse{
		ID:              rev.ID,
		TaskID:          rev.TaskID,
		MasterProfileID: rev.MasterProfileID,
		AuthorUserID:    rev.AuthorUserID,
		AuthorName:      rev.AuthorName,
		Rating:          rev.Rating,
		Comment:         rev.Comment,
		CreatedAt:       rev.CreatedAt,
		UpdatedAt:       rev.UpdatedAt,
	}
}

func (h *MasterHandler) handleDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, master.ErrInvalidUserID),
		errors.Is(err, master.ErrInvalidProfileID),
		errors.Is(err, master.ErrEmptyDisplayName),
		errors.Is(err, master.ErrDisplayNameTooLong),
		errors.Is(err, master.ErrCityTooLong),
		errors.Is(err, master.ErrTelegramTooLong),
		errors.Is(err, master.ErrInvalidAvatarURL),
		errors.Is(err, master.ErrEmptyContactPhone),
		errors.Is(err, master.ErrInvalidPhone),
		errors.Is(err, master.ErrDistrictTooLong),
		errors.Is(err, master.ErrEmptyTitle),
		errors.Is(err, master.ErrTitleTooShort),
		errors.Is(err, master.ErrTitleTooLong),
		errors.Is(err, master.ErrEmptyCategory),
		errors.Is(err, master.ErrInvalidCategory),
		errors.Is(err, master.ErrInvalidPrice):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})

	case errors.Is(err, master.ErrForbiddenAction):
		writeJSON(w, http.StatusForbidden, errorResponse{Error: err.Error()})

	case errors.Is(err, master.ErrProfileNotFound),
		errors.Is(err, master.ErrServiceNotFound):
		writeJSON(w, http.StatusNotFound, errorResponse{Error: err.Error()})

	case errors.Is(err, master.ErrProfileAlreadyExists),
		errors.Is(err, master.ErrServiceCategoryExists),
		errors.Is(err, master.ErrMaxActiveServicesReached),
		errors.Is(err, master.ErrProfileInUse):
		writeJSON(w, http.StatusConflict, errorResponse{Error: err.Error()})

	default:
		h.log.WithError(err).Error("unexpected internal error in master handler")
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
	}
}
