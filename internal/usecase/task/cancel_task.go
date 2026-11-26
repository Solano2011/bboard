package task

import (
	"context"
	"fmt"

	"bboard/internal/domain/task"

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

func (uc *CancelTaskUseCase) Execute(ctx context.Context, input ChangeTaskStatusInput) (*task.Task, error) {
	log := uc.log.WithFields(logrus.Fields{
		"task_id":  input.TaskID,
		"actor_id": input.ActorID,
	})

	t, err := uc.taskRepo.GetByID(ctx, input.TaskID)
	if err != nil {
		return nil, err
	}

	if err := t.Cancel(input.ActorID); err != nil {
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
