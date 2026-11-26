package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"bboard/internal/delivery/http/middleware"
	"bboard/internal/domain/review"
	"bboard/internal/domain/task"
	reviewUC "bboard/internal/usecase/review"

	"github.com/sirupsen/logrus"
)

type ReviewHandler struct {
	createUC *reviewUC.CreateReviewUseCase
	updateUC *reviewUC.UpdateReviewUseCase
	deleteUC *reviewUC.DeleteReviewUseCase
	log      *logrus.Entry
}

func NewReviewHandler(
	createUC *reviewUC.CreateReviewUseCase,
	updateUC *reviewUC.UpdateReviewUseCase,
	deleteUC *reviewUC.DeleteReviewUseCase,
	log *logrus.Logger,
) *ReviewHandler {
	return &ReviewHandler{
		createUC: createUC,
		updateUC: updateUC,
		deleteUC: deleteUC,
		log:      log.WithField("handler", "review"),
	}
}

type reviewRequest struct {
	Rating  int    `json:"rating"`
	Comment string `json:"comment"`
}

type reviewResponse struct {
	ID              int64     `json:"id"`
	TaskID          int64     `json:"task_id"`
	MasterProfileID int64     `json:"master_profile_id"`
	AuthorUserID    int64     `json:"author_user_id"`
	AuthorName      string    `json:"author_name"`
	Rating          int       `json:"rating"`
	Comment         string    `json:"comment"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Create обрабатывает POST /api/v1/tasks/{id}/review.
func (h *ReviewHandler) Create(w http.ResponseWriter, r *http.Request) {
	actorUserID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}

	taskID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || taskID <= 0 {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid task id in path"})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req reviewRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON payload"})
		return
	}

	created, err := h.createUC.Execute(r.Context(), reviewUC.CreateReviewInput{
		TaskID:      taskID,
		ActorUserID: actorUserID,
		Rating:      req.Rating,
		Comment:     req.Comment,
	})
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, mapReviewToResponse(created))
}

// Update обрабатывает PUT /api/v1/reviews/{id}.
func (h *ReviewHandler) Update(w http.ResponseWriter, r *http.Request) {
	actorUserID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}

	reviewID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || reviewID <= 0 {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid review id in path"})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req reviewRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON payload"})
		return
	}

	updated, err := h.updateUC.Execute(r.Context(), reviewUC.UpdateReviewInput{
		ReviewID:    reviewID,
		ActorUserID: actorUserID,
		Rating:      req.Rating,
		Comment:     req.Comment,
	})
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, mapReviewToResponse(updated))
}

// Delete обрабатывает DELETE /api/v1/reviews/{id}.
func (h *ReviewHandler) Delete(w http.ResponseWriter, r *http.Request) {
	actorUserID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}

	reviewID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || reviewID <= 0 {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid review id in path"})
		return
	}

	if err := h.deleteUC.Execute(r.Context(), reviewUC.DeleteReviewInput{
		ReviewID:    reviewID,
		ActorUserID: actorUserID,
	}); err != nil {
		h.handleDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *ReviewHandler) handleDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, review.ErrInvalidTaskID),
		errors.Is(err, review.ErrInvalidMasterID),
		errors.Is(err, review.ErrInvalidAuthorID),
		errors.Is(err, review.ErrInvalidRating):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})

	case errors.Is(err, review.ErrForbiddenReview):
		writeJSON(w, http.StatusForbidden, errorResponse{Error: err.Error()})

	case errors.Is(err, review.ErrReviewNotFound),
		errors.Is(err, task.ErrTaskNotFound):
		writeJSON(w, http.StatusNotFound, errorResponse{Error: err.Error()})

	case errors.Is(err, review.ErrReviewAlreadyExists),
		errors.Is(err, review.ErrTaskNotCompleted):
		writeJSON(w, http.StatusConflict, errorResponse{Error: err.Error()})

	default:
		h.log.WithError(err).Error("unexpected internal error in review handler")
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
	}
}
