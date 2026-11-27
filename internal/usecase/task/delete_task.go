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
	taskRepo     task.Repository
	photoStorage PhotoStorage
	log          *logrus.Entry
}

func NewDeleteTaskUseCase(
	taskRepo task.Repository,
	photoStorage PhotoStorage,
	log *logrus.Logger,
) *DeleteTaskUseCase {
	return &DeleteTaskUseCase{
		taskRepo:     taskRepo,
		photoStorage: photoStorage,
		log:          log.WithField("usecase", "task.delete"),
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

	storageKeys, err := uc.taskRepo.Delete(ctx, t.ID, expectedStatuses...)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}

	if len(storageKeys) > 0 && uc.photoStorage != nil {
		if err := uc.photoStorage.DeleteTaskDir(ctx, t.ID); err != nil {
			uc.log.WithError(err).WithField("task_id", t.ID).Warn("failed to remove task photos directory from disk")
		}
	}

	uc.log.WithFields(logrus.Fields{
		"task_id":    t.ID,
		"actor_id":   input.ActorID,
		"actor_role": input.ActorRole,
	}).Info("task successfully deleted")
	return nil
}
