package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"bboard/internal/delivery/http/middleware"
	"bboard/internal/domain/task"
	taskUC "bboard/internal/usecase/task"

	"github.com/sirupsen/logrus"
)

// TaskHandler обрабатывает HTTP-запросы, связанные с объявлениями заказчиков.
type TaskHandler struct {
	createUC   *taskUC.CreateTaskUseCase
	listUC     *taskUC.ListTasksUseCase
	assignUC   *taskUC.AssignMasterUseCase
	completeUC *taskUC.CompleteTaskUseCase
	cancelUC   *taskUC.CancelTaskUseCase
	log        *logrus.Entry
}

// NewTaskHandler создаёт новый экземпляр HTTP-хэндлера объявлений.
func NewTaskHandler(
	createUC *taskUC.CreateTaskUseCase,
	listUC *taskUC.ListTasksUseCase,
	assignUC *taskUC.AssignMasterUseCase,
	completeUC *taskUC.CompleteTaskUseCase,
	cancelUC *taskUC.CancelTaskUseCase,
	log *logrus.Logger,
) *TaskHandler {
	return &TaskHandler{
		createUC:   createUC,
		listUC:     listUC,
		assignUC:   assignUC,
		completeUC: completeUC,
		cancelUC:   cancelUC,
		log:        log.WithField("handler", "task"),
	}
}

type createTaskRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Category    string `json:"category"`
	City        string `json:"city"`
	Budget      int64  `json:"budget"`
}

type assignMasterRequest struct {
	MasterID int64 `json:"master_id"`
}

type taskResponse struct {
	ID          int64       `json:"id"`
	AuthorID    int64       `json:"author_id"`
	MasterID    *int64      `json:"master_id,omitempty"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Category    string      `json:"category"`
	City        string      `json:"city,omitempty"`
	Budget      int64       `json:"budget"`
	Status      task.Status `json:"status"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

// Create обрабатывает POST /api/v1/tasks.
func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	authorID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req createTaskRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&req); err != nil {
		h.log.WithError(err).Warn("failed to decode create task request")
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON payload"})
		return
	}

	created, err := h.createUC.Execute(r.Context(), taskUC.CreateTaskInput{
		AuthorID:    authorID,
		Title:       req.Title,
		Description: req.Description,
		Category:    req.Category,
		City:        req.City,
		Budget:      req.Budget,
	})
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, mapTaskToResponse(created))
}

// List обрабатывает GET /api/v1/tasks.
func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))

	tasks, err := h.listUC.Execute(r.Context(), taskUC.ListTasksInput{
		Category: q.Get("category"),
		City:     q.Get("city"),
		Status:   task.Status(q.Get("status")),
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	resp := make([]taskResponse, 0, len(tasks))
	for _, t := range tasks {
		resp = append(resp, mapTaskToResponse(t))
	}

	writeJSON(w, http.StatusOK, resp)
}

// AssignMaster обрабатывает POST /api/v1/tasks/{id}/assign.
func (h *TaskHandler) AssignMaster(w http.ResponseWriter, r *http.Request) {
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

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req assignMasterRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&req); err != nil {
		h.log.WithError(err).Warn("failed to decode assign master request")
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON payload"})
		return
	}

	updated, err := h.assignUC.Execute(r.Context(), taskUC.AssignMasterInput{
		TaskID:   taskID,
		ActorID:  actorID,
		MasterID: req.MasterID,
	})
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, mapTaskToResponse(updated))
}

// Complete обрабатывает POST /api/v1/tasks/{id}/complete.
func (h *TaskHandler) Complete(w http.ResponseWriter, r *http.Request) {
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

	updated, err := h.completeUC.Execute(r.Context(), taskUC.ChangeTaskStatusInput{
		TaskID:  taskID,
		ActorID: actorID,
	})
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, mapTaskToResponse(updated))
}

// Cancel обрабатывает POST /api/v1/tasks/{id}/cancel.
func (h *TaskHandler) Cancel(w http.ResponseWriter, r *http.Request) {
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

	updated, err := h.cancelUC.Execute(r.Context(), taskUC.ChangeTaskStatusInput{
		TaskID:  taskID,
		ActorID: actorID,
	})
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, mapTaskToResponse(updated))
}

func mapTaskToResponse(t *task.Task) taskResponse {
	return taskResponse{
		ID:          t.ID,
		AuthorID:    t.AuthorID,
		MasterID:    t.MasterID,
		Title:       t.Title,
		Description: t.Description,
		Category:    t.Category,
		City:        t.City,
		Budget:      t.Budget,
		Status:      t.Status,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}

func (h *TaskHandler) handleDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, task.ErrEmptyTitle),
		errors.Is(err, task.ErrTitleTooShort),
		errors.Is(err, task.ErrEmptyDescription),
		errors.Is(err, task.ErrEmptyCategory),
		errors.Is(err, task.ErrInvalidBudget),
		errors.Is(err, task.ErrInvalidAuthor),
		errors.Is(err, task.ErrInvalidMaster),
		errors.Is(err, task.ErrSelfAssignment):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})

	case errors.Is(err, task.ErrForbiddenAction):
		writeJSON(w, http.StatusForbidden, errorResponse{Error: err.Error()})

	case errors.Is(err, task.ErrTaskNotFound):
		writeJSON(w, http.StatusNotFound, errorResponse{Error: err.Error()})

	case errors.Is(err, task.ErrInvalidStatus):
		writeJSON(w, http.StatusConflict, errorResponse{Error: err.Error()})

	default:
		h.log.WithError(err).Error("unexpected internal error in task handler")
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
	}
}
