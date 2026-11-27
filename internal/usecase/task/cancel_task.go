package task

import (
	"context"
	"fmt"

	"bboard/internal/domain/task"
	"bboard/internal/domain/user"

	"github.com/sirupsen/logrus"
)

type CancelTaskUseCase struct {
	taskRepo task.Repository
	log      *logrus.Entry
}

func NewCancelTaskUseCase(taskRepo task.Repository, log *logrus.Logger) *CancelTaskUseCase {
	return &CancelTaskUseCase{
		taskRepo: taskRepo,
		log:      log.WithField("usecase", "task.cancel"),
	}
}

func (uc *CancelTaskUseCase) Execute(ctx context.Context, taskID, actorID int64, actorRole user.Role) (*task.Task, error) {
	log := uc.log.WithFields(logrus.Fields{
		"task_id":    taskID,
		"actor_id":   actorID,
		"actor_role": actorRole,
	})

	t, err := uc.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	if err := t.Cancel(actorID, actorRole); err != nil {
		log.WithError(err).Warn("cancel task rejected by domain rules")
		return nil, err
	}

	if err := uc.taskRepo.Update(ctx, t, task.StatusOpen, task.StatusInProgress, task.StatusPendingCompletion); err != nil {
		log.WithError(err).Error("failed to update task status to canceled")
		return nil, fmt.Errorf("update task: %w", err)
	}

	log.Info("task marked as canceled")
	return t, nil
}
