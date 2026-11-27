package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"bboard/internal/delivery/http/middleware"
	"bboard/internal/domain/master"
	"bboard/internal/domain/review"
	"bboard/internal/domain/task"
	"bboard/internal/domain/user"
	taskUC "bboard/internal/usecase/task"

	"github.com/sirupsen/logrus"
)

// TaskHandler обрабатывает HTTP-запросы, связанные с объявлениями заказчиков.
type TaskHandler struct {
	createUC        *taskUC.CreateTaskUseCase
	listUC          *taskUC.ListTasksUseCase
	updateUC        *taskUC.UpdateTaskUseCase
	deleteUC        *taskUC.DeleteTaskUseCase
	assignUC        *taskUC.AssignMasterUseCase
	markCompletedUC *taskUC.MarkTaskCompletedUseCase
	completeUC      *taskUC.CompleteTaskUseCase
	cancelUC        *taskUC.CancelTaskUseCase
	uploadPhotosUC  *taskUC.UploadTaskPhotosUseCase
	deletePhotoUC   *taskUC.DeleteTaskPhotoUseCase
	taskRepo        task.Repository
	masterRepo      master.Repository
	reviewRepo      review.Repository
	log             *logrus.Entry
}

// NewTaskHandler создаёт новый экземпляр HTTP-хэндлера объявлений.
func NewTaskHandler(
	createUC *taskUC.CreateTaskUseCase,
	listUC *taskUC.ListTasksUseCase,
	updateUC *taskUC.UpdateTaskUseCase,
	deleteUC *taskUC.DeleteTaskUseCase,
	assignUC *taskUC.AssignMasterUseCase,
	markCompletedUC *taskUC.MarkTaskCompletedUseCase,
	completeUC *taskUC.CompleteTaskUseCase,
	cancelUC *taskUC.CancelTaskUseCase,
	uploadPhotosUC *taskUC.UploadTaskPhotosUseCase,
	deletePhotoUC *taskUC.DeleteTaskPhotoUseCase,
	taskRepo task.Repository,
	masterRepo master.Repository,
	reviewRepo review.Repository,
	log *logrus.Logger,
) *TaskHandler {
	return &TaskHandler{
		createUC:        createUC,
		listUC:          listUC,
		updateUC:        updateUC,
		deleteUC:        deleteUC,
		assignUC:        assignUC,
		markCompletedUC: markCompletedUC,
		completeUC:      completeUC,
		cancelUC:        cancelUC,
		uploadPhotosUC:  uploadPhotosUC,
		deletePhotoUC:   deletePhotoUC,
		taskRepo:        taskRepo,
		masterRepo:      masterRepo,
		reviewRepo:      reviewRepo,
		log:             log.WithField("handler", "task"),
	}
}

type createTaskRequest struct {
	Title        string `json:"title"`
	Description  string `json:"description"`
	Category     string `json:"category"`
	City         string `json:"city"`
	District     string `json:"district"`
	ContactPhone string `json:"contact_phone"`
	WorkAddress  string `json:"work_address"`
	Budget       int64  `json:"budget"`
}

type assignMasterRequest struct {
	MasterProfileID int64 `json:"master_profile_id"`
}

type taskPhotoResponse struct {
	ID        int64     `json:"id"`
	URL       string    `json:"url"`
	MimeType  string    `json:"mime_type"`
	SizeBytes int64     `json:"size_bytes"`
	CreatedAt time.Time `json:"created_at"`
}

type taskResponse struct {
	ID               int64               `json:"id"`
	AuthorID         int64               `json:"author_id"`
	AssignedMasterID *int64              `json:"assigned_master_id,omitempty"`
	IsAssignedToMe   bool                `json:"is_assigned_to_me,omitempty"`
	Title            string              `json:"title"`
	Description      string              `json:"description"`
	Category         string              `json:"category"`
	City             string              `json:"city,omitempty"`
	District         string              `json:"district,omitempty"`
	ContactPhone     string              `json:"contact_phone"`
	WorkAddress      string              `json:"work_address"`
	Budget           int64               `json:"budget"`
	Status           task.Status         `json:"status"`
	Photos           []taskPhotoResponse `json:"photos"`
	MyReview         *reviewResponse     `json:"my_review,omitempty"`
	CreatedAt        time.Time           `json:"created_at"`
	UpdatedAt        time.Time           `json:"updated_at"`
}

// Create обрабатывает POST /api/v1/tasks.
func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	authorID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}
	actorRole, _ := middleware.UserRoleFromContext(r.Context())

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
		AuthorID:     authorID,
		Title:        req.Title,
		Description:  req.Description,
		Category:     req.Category,
		City:         req.City,
		District:     req.District,
		ContactPhone: req.ContactPhone,
		WorkAddress:  req.WorkAddress,
		Budget:       req.Budget,
	})
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, h.mapTaskToResponse(r.Context(), created, authorID, actorRole, 0, nil, nil))
}

// Update обрабатывает PUT /api/v1/tasks/{id}.
func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	actorID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}
	actorRole, _ := middleware.UserRoleFromContext(r.Context())

	taskID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || taskID <= 0 {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid task id"})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req createTaskRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&req); err != nil {
		h.log.WithError(err).Warn("failed to decode update task request")
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON payload"})
		return
	}

	updated, err := h.updateUC.Execute(r.Context(), taskUC.UpdateTaskInput{
		TaskID:       taskID,
		ActorID:      actorID,
		ActorRole:    actorRole,
		Title:        req.Title,
		Description:  req.Description,
		Category:     req.Category,
		City:         req.City,
		District:     req.District,
		ContactPhone: req.ContactPhone,
		WorkAddress:  req.WorkAddress,
		Budget:       req.Budget,
	})
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	photos, _ := h.taskRepo.ListPhotosByTaskID(r.Context(), updated.ID)
	writeJSON(w, http.StatusOK, h.mapTaskToResponse(r.Context(), updated, actorID, actorRole, 0, nil, photos))
}

// Delete обрабатывает DELETE /api/v1/tasks/{id}.
func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	actorID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}
	actorRole, _ := middleware.UserRoleFromContext(r.Context())

	taskID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || taskID <= 0 {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid task id in path"})
		return
	}

	if err := h.deleteUC.Execute(r.Context(), taskUC.DeleteTaskInput{
		TaskID:    taskID,
		ActorID:   actorID,
		ActorRole: actorRole,
	}); err != nil {
		h.handleDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// List обрабатывает GET /api/v1/tasks.
func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
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

	status := task.Status(q.Get("status"))
	if status != "" &&
		status != task.StatusOpen &&
		status != task.StatusInProgress &&
		status != task.StatusPendingCompletion &&
		status != task.StatusCompleted &&
		status != task.StatusCanceled {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid status query parameter"})
		return
	}

	tasks, err := h.listUC.Execute(r.Context(), taskUC.ListTasksInput{
		Category: q.Get("category"),
		City:     q.Get("city"),
		Status:   status,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	currentUserID, _ := middleware.UserIDFromContext(r.Context())
	currentUserRole, _ := middleware.UserRoleFromContext(r.Context())
	currentUserProfileID := h.resolveProfileID(r.Context(), currentUserID)

	taskIDs := make([]int64, 0, len(tasks))
	for _, t := range tasks {
		taskIDs = append(taskIDs, t.ID)
	}

	photosByTaskID, err := h.taskRepo.ListPhotosByTaskIDs(r.Context(), taskIDs)
	if err != nil {
		h.log.WithError(err).Warn("failed to batch-load task photos")
	}

	var completedAuthorTaskIDs []int64
	if currentUserID > 0 {
		for _, t := range tasks {
			if t.AuthorID == currentUserID && t.Status == task.StatusCompleted {
				completedAuthorTaskIDs = append(completedAuthorTaskIDs, t.ID)
			}
		}
	}

	var reviewsByTaskID map[int64]*review.Review
	if len(completedAuthorTaskIDs) > 0 {
		reviewsByTaskID, err = h.reviewRepo.GetByTaskIDs(r.Context(), completedAuthorTaskIDs)
		if err != nil {
			h.log.WithError(err).Warn("failed to batch-load task reviews")
		}
	}

	resp := make([]taskResponse, 0, len(tasks))
	for _, t := range tasks {
		var preloadedReview *review.Review
		if reviewsByTaskID != nil {
			preloadedReview = reviewsByTaskID[t.ID]
		}
		var taskPhotos []*task.Photo
		if photosByTaskID != nil {
			taskPhotos = photosByTaskID[t.ID]
		}
		resp = append(resp, h.mapTaskToResponse(r.Context(), t, currentUserID, currentUserRole, currentUserProfileID, preloadedReview, taskPhotos))
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
		TaskID:          taskID,
		ActorID:         actorID,
		MasterProfileID: req.MasterProfileID,
	})
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	actorRole, _ := middleware.UserRoleFromContext(r.Context())
	photos, _ := h.taskRepo.ListPhotosByTaskID(r.Context(), updated.ID)
	writeJSON(w, http.StatusOK, h.mapTaskToResponse(r.Context(), updated, actorID, actorRole, 0, nil, photos))
}

// MarkCompleted обрабатывает POST /api/v1/tasks/{id}/mark-completed (мастер отмечает выполнение).
func (h *TaskHandler) MarkCompleted(w http.ResponseWriter, r *http.Request) {
	actorID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}
	actorRole, _ := middleware.UserRoleFromContext(r.Context())

	taskID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || taskID <= 0 {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid task id in path"})
		return
	}

	updated, err := h.markCompletedUC.Execute(r.Context(), taskUC.ChangeTaskStatusInput{
		TaskID:  taskID,
		ActorID: actorID,
	})
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	photos, _ := h.taskRepo.ListPhotosByTaskID(r.Context(), updated.ID)
	writeJSON(w, http.StatusOK, h.mapTaskToResponse(r.Context(), updated, actorID, actorRole, h.resolveProfileID(r.Context(), actorID), nil, photos))
}

// Complete обрабатывает POST /api/v1/tasks/{id}/complete.
func (h *TaskHandler) Complete(w http.ResponseWriter, r *http.Request) {
	actorID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}
	actorRole, _ := middleware.UserRoleFromContext(r.Context())

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

	photos, _ := h.taskRepo.ListPhotosByTaskID(r.Context(), updated.ID)
	writeJSON(w, http.StatusOK, h.mapTaskToResponse(r.Context(), updated, actorID, actorRole, 0, nil, photos))
}

// Cancel обрабатывает POST /api/v1/tasks/{id}/cancel.
func (h *TaskHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	actorID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}
	actorRole, _ := middleware.UserRoleFromContext(r.Context())

	taskID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || taskID <= 0 {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid task id in path"})
		return
	}

	updated, err := h.cancelUC.Execute(r.Context(), taskID, actorID, actorRole)
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	photos, _ := h.taskRepo.ListPhotosByTaskID(r.Context(), updated.ID)
	writeJSON(w, http.StatusOK, h.mapTaskToResponse(r.Context(), updated, actorID, actorRole, 0, nil, photos))
}

// UploadPhotos обрабатывает POST /api/v1/tasks/{id}/photos.
func (h *TaskHandler) UploadPhotos(w http.ResponseWriter, r *http.Request) {
	actorID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}
	actorRole, _ := middleware.UserRoleFromContext(r.Context())

	taskID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || taskID <= 0 {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid task id in path"})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 16<<20)
	defer func() {
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}
	}()

	if err := r.ParseMultipartForm(4 << 20); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			writeJSON(w, http.StatusRequestEntityTooLarge, errorResponse{Error: "request body exceeds maximum upload size"})
			return
		}
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid multipart form payload"})
		return
	}

	headers := r.MultipartForm.File["photos"]
	if len(headers) == 0 {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: task.ErrEmptyPhotoFile.Error()})
		return
	}
	if len(headers) > task.MaxPhotosPerTask {
		writeJSON(w, http.StatusConflict, errorResponse{Error: task.ErrTooManyPhotos.Error()})
		return
	}

	files := make([]taskUC.UploadedPhotoFile, 0, len(headers))
	for _, fh := range headers {
		f, err := fh.Open()
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "failed to open uploaded file"})
			return
		}
		data, readErr := io.ReadAll(io.LimitReader(f, task.MaxPhotoSizeBytes+1))
		_ = f.Close()
		if readErr != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "failed to read uploaded file"})
			return
		}
		if len(data) > task.MaxPhotoSizeBytes {
			writeJSON(w, http.StatusRequestEntityTooLarge, errorResponse{Error: task.ErrPhotoTooLarge.Error()})
			return
		}
		files = append(files, taskUC.UploadedPhotoFile{Data: data})
	}

	photos, err := h.uploadPhotosUC.Execute(r.Context(), taskUC.UploadTaskPhotosInput{
		TaskID:    taskID,
		ActorID:   actorID,
		ActorRole: actorRole,
		Files:     files,
	})
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, mapPhotosToResponse(photos))
}

// DeletePhoto обрабатывает DELETE /api/v1/tasks/{id}/photos/{photoId}.
func (h *TaskHandler) DeletePhoto(w http.ResponseWriter, r *http.Request) {
	actorID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}
	actorRole, _ := middleware.UserRoleFromContext(r.Context())

	taskID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || taskID <= 0 {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid task id in path"})
		return
	}
	photoID, err := strconv.ParseInt(r.PathValue("photoId"), 10, 64)
	if err != nil || photoID <= 0 {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid photo id in path"})
		return
	}

	if err := h.deletePhotoUC.Execute(r.Context(), taskUC.DeleteTaskPhotoInput{
		TaskID:    taskID,
		PhotoID:   photoID,
		ActorID:   actorID,
		ActorRole: actorRole,
	}); err != nil {
		h.handleDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *TaskHandler) resolveProfileID(ctx context.Context, userID int64) int64 {
	if userID <= 0 {
		return 0
	}
	profile, err := h.masterRepo.GetProfileByUserID(ctx, userID)
	if err != nil {
		return 0
	}
	return profile.ID
}

func mapPhotosToResponse(photos []*task.Photo) []taskPhotoResponse {
	resp := make([]taskPhotoResponse, 0, len(photos))
	for _, p := range photos {
		resp = append(resp, taskPhotoResponse{
			ID:        p.ID,
			URL:       "/uploads/" + p.StorageKey,
			MimeType:  p.MimeType,
			SizeBytes: p.SizeBytes,
			CreatedAt: p.CreatedAt,
		})
	}
	return resp
}

func (h *TaskHandler) mapTaskToResponse(ctx context.Context, t *task.Task, currentUserID int64, currentUserRole user.Role, currentUserProfileID int64, preloadedReview *review.Review, photos []*task.Photo) taskResponse {
	isAssignedToMe := currentUserProfileID > 0 &&
		t.AssignedMasterID != nil &&
		*t.AssignedMasterID == currentUserProfileID

	resp := taskResponse{
		ID:               t.ID,
		AuthorID:         t.AuthorID,
		AssignedMasterID: t.AssignedMasterID,
		IsAssignedToMe:   isAssignedToMe,
		Title:            t.Title,
		Description:      t.Description,
		Category:         t.Category,
		City:             t.City,
		District:         t.District,
		Budget:           t.Budget,
		Status:           t.Status,
		Photos:           mapPhotosToResponse(photos),
		CreatedAt:        t.CreatedAt,
		UpdatedAt:        t.UpdatedAt,
	}

	isAuthor := currentUserID > 0 && t.AuthorID == currentUserID
	isAssignedWithContacts := isAssignedToMe &&
		(t.Status == task.StatusInProgress || t.Status == task.StatusPendingCompletion || t.Status == task.StatusCompleted)

	if isAuthor || isAssignedWithContacts || currentUserRole.IsModeratorOrAdmin() {
		resp.ContactPhone = t.ContactPhone
		resp.WorkAddress = t.WorkAddress
	}

	if isAuthor && t.Status == task.StatusCompleted && preloadedReview != nil {
		mapped := mapReviewToResponse(preloadedReview)
		resp.MyReview = &mapped
	}

	return resp
}

func (h *TaskHandler) handleDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, task.ErrEmptyTitle),
		errors.Is(err, task.ErrTitleTooShort),
		errors.Is(err, task.ErrEmptyDescription),
		errors.Is(err, task.ErrEmptyCategory),
		errors.Is(err, task.ErrInvalidCategory),
		errors.Is(err, task.ErrInvalidBudget),
		errors.Is(err, task.ErrInvalidAuthor),
		errors.Is(err, task.ErrInvalidMaster),
		errors.Is(err, task.ErrSelfAssignment),
		errors.Is(err, task.ErrDistrictTooLong),
		errors.Is(err, task.ErrInvalidPhone),
		errors.Is(err, task.ErrAddressTooLong),
		errors.Is(err, task.ErrMasterNotAssigned),
		errors.Is(err, task.ErrEmptyPhotoFile),
		errors.Is(err, task.ErrUnsupportedPhotoType):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})

	case errors.Is(err, task.ErrPhotoTooLarge):
		writeJSON(w, http.StatusRequestEntityTooLarge, errorResponse{Error: err.Error()})

	case errors.Is(err, task.ErrForbiddenAction):
		writeJSON(w, http.StatusForbidden, errorResponse{Error: err.Error()})

	case errors.Is(err, task.ErrTaskNotFound),
		errors.Is(err, task.ErrPhotoNotFound):
		writeJSON(w, http.StatusNotFound, errorResponse{Error: err.Error()})

	case errors.Is(err, task.ErrInvalidStatus),
		errors.Is(err, task.ErrTooManyPhotos):
		writeJSON(w, http.StatusConflict, errorResponse{Error: err.Error()})

	default:
		h.log.WithError(err).Error("unexpected internal error in task handler")
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
	}
}
