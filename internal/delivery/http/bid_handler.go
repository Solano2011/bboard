package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"bboard/internal/delivery/http/middleware"
	"bboard/internal/domain/bid"
	"bboard/internal/domain/task"
	bidUC "bboard/internal/usecase/bid"

	"github.com/sirupsen/logrus"
)

type BidHandler struct {
	createUC *bidUC.CreateBidUseCase
	listUC   *bidUC.ListBidsUseCase
	log      *logrus.Entry
}

func NewBidHandler(
	createUC *bidUC.CreateBidUseCase,
	listUC *bidUC.ListBidsUseCase,
	log *logrus.Logger,
) *BidHandler {
	return &BidHandler{
		createUC: createUC,
		listUC:   listUC,
		log:      log.WithField("handler", "bid"),
	}
}

type createBidRequest struct {
	ProposedPrice int64  `json:"proposed_price"`
	Comment       string `json:"comment"`
}

type bidResponse struct {
	ID            int64     `json:"id"`
	TaskID        int64     `json:"task_id"`
	MasterID      int64     `json:"master_id"`
	ProposedPrice int64     `json:"proposed_price"`
	Comment       string    `json:"comment"`
	CreatedAt     time.Time `json:"created_at"`
}

// Create обрабатывает POST /api/v1/tasks/{id}/bids (отклик мастера на задачу).
func (h *BidHandler) Create(w http.ResponseWriter, r *http.Request) {
	masterID, ok := middleware.UserIDFromContext(r.Context())
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

	var req createBidRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&req); err != nil {
		h.log.WithError(err).Warn("failed to decode create bid request")
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON payload"})
		return
	}

	created, err := h.createUC.Execute(r.Context(), bidUC.CreateBidInput{
		TaskID:        taskID,
		MasterID:      masterID,
		ProposedPrice: req.ProposedPrice,
		Comment:       req.Comment,
	})
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, mapBidToResponse(created))
}

// ListByTask обрабатывает GET /api/v1/tasks/{id}/bids (список откликов для автора задачи).
func (h *BidHandler) ListByTask(w http.ResponseWriter, r *http.Request) {
	actorID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}

	taskID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || taskID <= 0 {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid task id in path"})
		return
	}

	bids, err := h.listUC.Execute(r.Context(), bidUC.ListBidsInput{
		TaskID:  taskID,
		ActorID: actorID,
	})
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	resp := make([]bidResponse, 0, len(bids))
	for _, b := range bids {
		resp = append(resp, mapBidToResponse(b))
	}

	writeJSON(w, http.StatusOK, resp)
}

func mapBidToResponse(b *bid.Bid) bidResponse {
	return bidResponse{
		ID:            b.ID,
		TaskID:        b.TaskID,
		MasterID:      b.MasterID,
		ProposedPrice: b.ProposedPrice,
		Comment:       b.Comment,
		CreatedAt:     b.CreatedAt,
	}
}

func (h *BidHandler) handleDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, bid.ErrInvalidTaskID),
		errors.Is(err, bid.ErrInvalidMasterID),
		errors.Is(err, bid.ErrInvalidPrice),
		errors.Is(err, bid.ErrEmptyComment),
		errors.Is(err, bid.ErrCommentTooShort),
		errors.Is(err, bid.ErrSelfBidNotAllowed):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})

	case errors.Is(err, bid.ErrMasterProfileNeeded),
		errors.Is(err, bid.ErrForbiddenView):
		writeJSON(w, http.StatusForbidden, errorResponse{Error: err.Error()})

	case errors.Is(err, task.ErrTaskNotFound):
		writeJSON(w, http.StatusNotFound, errorResponse{Error: err.Error()})

	case errors.Is(err, bid.ErrBidAlreadyExists),
		errors.Is(err, bid.ErrTaskNotOpenForBids):
		writeJSON(w, http.StatusConflict, errorResponse{Error: err.Error()})

	default:
		h.log.WithError(err).Error("unexpected internal error in bid handler")
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
	}
}
