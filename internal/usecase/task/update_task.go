package task

import (
	"context"
	"fmt"

	"bboard/internal/domain/task"
	"bboard/internal/domain/user"

	"github.com/sirupsen/logrus"
)

// UpdateTaskInput — входные данные для редактирования объявления заказчика.
type UpdateTaskInput struct {
	TaskID       int64
	ActorID      int64
	ActorRole    user.Role
	Title        string
	Description  string
	Category     string
	City         string
	District     string
	ContactPhone string
	WorkAddress  string
	Budget       int64
}

// UpdateTaskUseCase реализует бизнес-сценарий редактирования объявления.
type UpdateTaskUseCase struct {
	taskRepo task.Repository
	log      *logrus.Entry
}

// NewUpdateTaskUseCase собирает зависимости для сценария редактирования объявления.
func NewUpdateTaskUseCase(taskRepo task.Repository, log *logrus.Logger) *UpdateTaskUseCase {
	return &UpdateTaskUseCase{
		taskRepo: taskRepo,
		log:      log.WithField("usecase", "task.update"),
	}
}

// Execute проверяет права автора или модератора, статус задачи, валидирует поля и сохраняет изменения.
func (uc *UpdateTaskUseCase) Execute(ctx context.Context, input UpdateTaskInput) (*task.Task, error) {
	if input.TaskID <= 0 {
		return nil, task.ErrTaskNotFound
	}

	t, err := uc.taskRepo.GetByID(ctx, input.TaskID)
	if err != nil {
		return nil, err
	}

	if err := t.Update(
		input.ActorID,
		input.ActorRole,
		input.Title,
		input.Description,
		input.Category,
		input.City,
		input.District,
		input.ContactPhone,
		input.WorkAddress,
		input.Budget,
	); err != nil {
		uc.log.WithFields(logrus.Fields{
			"task_id":    input.TaskID,
			"actor_id":   input.ActorID,
			"actor_role": input.ActorRole,
		}).WithError(err).Warn("update task rejected")
		return nil, err
	}

	allowedPrevStatuses := []task.Status{task.StatusOpen}
	if input.ActorRole.IsModeratorOrAdmin() {
		allowedPrevStatuses = []task.Status{
			task.StatusOpen,
			task.StatusInProgress,
			task.StatusPendingCompletion,
		}
	}

	if err := uc.taskRepo.Update(ctx, t, allowedPrevStatuses...); err != nil {
		uc.log.WithError(err).Error("failed to update task")
		return nil, fmt.Errorf("update task: %w", err)
	}

	uc.log.WithFields(logrus.Fields{
		"task_id":    t.ID,
		"actor_id":   input.ActorID,
		"actor_role": input.ActorRole,
	}).Info("task successfully updated")

	return t, nil
}
