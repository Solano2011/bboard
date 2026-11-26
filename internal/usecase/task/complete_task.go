package task

import (
	"context"
	"fmt"

	"bboard/internal/domain/task"

	"github.com/sirupsen/logrus"
)

type ChangeTaskStatusInput struct {
	TaskID  int64
	ActorID int64
}

type CompleteTaskUseCase struct {
	taskRepo task.Repository
	log      *logrus.Entry
}

func NewCompleteTaskUseCase(taskRepo task.Repository, log *logrus.Logger) *CompleteTaskUseCase {
	return &CompleteTaskUseCase{
		taskRepo: taskRepo,
		log:      log.WithField("usecase", "task.complete"),
	}
}

func (uc *CompleteTaskUseCase) Execute(ctx context.Context, input ChangeTaskStatusInput) (*task.Task, error) {
	log := uc.log.WithFields(logrus.Fields{
		"task_id":  input.TaskID,
		"actor_id": input.ActorID,
	})

	t, err := uc.taskRepo.GetByID(ctx, input.TaskID)
	if err != nil {
		return nil, err
	}

	if err := t.Complete(input.ActorID); err != nil {
		log.WithError(err).Warn("complete task rejected by domain rules")
		return nil, err
	}

	if err := uc.taskRepo.CompleteAndIncrementMasterCounter(ctx, t); err != nil {
		log.WithError(err).Error("failed to complete task and increment master counter")
		return nil, fmt.Errorf("complete task: %w", err)
	}

	log.Info("task marked as completed")
	return t, nil
}
