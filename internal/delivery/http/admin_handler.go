package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"bboard/internal/delivery/http/middleware"
	"bboard/internal/domain/user"
	adminUC "bboard/internal/usecase/admin"

	"github.com/sirupsen/logrus"
)

type AdminHandler struct {
	listUsersUC      *adminUC.ListUsersUseCase
	updateUserRoleUC *adminUC.UpdateUserRoleUseCase
	log              *logrus.Entry
}

func NewAdminHandler(
	listUsersUC *adminUC.ListUsersUseCase,
	updateUserRoleUC *adminUC.UpdateUserRoleUseCase,
	log *logrus.Logger,
) *AdminHandler {
	return &AdminHandler{
		listUsersUC:      listUsersUC,
		updateUserRoleUC: updateUserRoleUC,
		log:              log.WithField("handler", "admin"),
	}
}

type updateUserRoleRequest struct {
	Role user.Role `json:"role"`
}

// ListUsers обрабатывает GET /api/v1/admin/users.
func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	_, ok := middleware.UserIDFromContext(r.Context())
	actorRole, roleOK := middleware.UserRoleFromContext(r.Context())
	if !ok || !roleOK {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}

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

	users, err := h.listUsersUC.Execute(r.Context(), adminUC.ListUsersInput{
		ActorRole: actorRole,
		Search:    q.Get("search"),
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	resp := make([]userResponse, 0, len(users))
	for _, u := range users {
		resp = append(resp, mapUserToResponse(u))
	}

	writeJSON(w, http.StatusOK, resp)
}

// UpdateUserRole обрабатывает PATCH /api/v1/admin/users/{id}/role.
func (h *AdminHandler) UpdateUserRole(w http.ResponseWriter, r *http.Request) {
	actorID, ok := middleware.UserIDFromContext(r.Context())
	actorRole, roleOK := middleware.UserRoleFromContext(r.Context())
	if !ok || !roleOK {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}

	targetUserID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || targetUserID <= 0 {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid user id in path"})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req updateUserRoleRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON payload"})
		return
	}

	updated, err := h.updateUserRoleUC.Execute(r.Context(), adminUC.UpdateUserRoleInput{
		ActorID:      actorID,
		ActorRole:    actorRole,
		TargetUserID: targetUserID,
		NewRole:      req.Role,
	})
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, mapUserToResponse(updated))
}

func (h *AdminHandler) handleDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, user.ErrInvalidRole):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})

	case errors.Is(err, user.ErrForbiddenRoleChange),
		errors.Is(err, user.ErrCannotChangeOwnRole):
		writeJSON(w, http.StatusForbidden, errorResponse{Error: err.Error()})

	case errors.Is(err, user.ErrUserNotFound):
		writeJSON(w, http.StatusNotFound, errorResponse{Error: err.Error()})

	default:
		h.log.WithError(err).Error("unexpected internal error in admin handler")
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
	}
}
