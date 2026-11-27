package task

import (
	"context"
	"fmt"

	"bboard/internal/domain/task"
	"bboard/internal/domain/user"

	"github.com/sirupsen/logrus"
)

type DeleteTaskInput struct {
	TaskID    int64
	ActorID   int64
	ActorRole user.Role
}

type DeleteTaskUseCase struct {
	taskRepo task.Repository
	log      *logrus.Entry
}

func NewDeleteTaskUseCase(taskRepo task.Repository, log *logrus.Logger) *DeleteTaskUseCase {
	return &DeleteTaskUseCase{
		taskRepo: taskRepo,
		log:      log.WithField("usecase", "task.delete"),
	}
}

func (uc *DeleteTaskUseCase) Execute(ctx context.Context, input DeleteTaskInput) error {
	if input.TaskID <= 0 {
		return task.ErrTaskNotFound
	}

	t, err := uc.taskRepo.GetByID(ctx, input.TaskID)
	if err != nil {
		return err
	}

	if err := t.CanDelete(input.ActorID, input.ActorRole); err != nil {
		uc.log.WithFields(logrus.Fields{
			"task_id":    input.TaskID,
			"actor_id":   input.ActorID,
			"actor_role": input.ActorRole,
		}).WithError(err).Warn("delete task rejected")
		return err
	}

	var expectedStatuses []task.Status
	if !input.ActorRole.IsModeratorOrAdmin() {
		expectedStatuses = []task.Status{task.StatusOpen}
	}

	if err := uc.taskRepo.Delete(ctx, t.ID, expectedStatuses...); err != nil {
		return fmt.Errorf("delete task: %w", err)
	}

	uc.log.WithFields(logrus.Fields{
		"task_id":    t.ID,
		"actor_id":   input.ActorID,
		"actor_role": input.ActorRole,
	}).Info("task successfully deleted")
	return nil
}
