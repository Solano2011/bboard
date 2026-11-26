package task

import (
	"context"
	"errors"
	"fmt"

	"bboard/internal/domain/master"
	"bboard/internal/domain/task"

	"github.com/sirupsen/logrus"
)

type MarkTaskCompletedUseCase struct {
	taskRepo   task.Repository
	masterRepo master.Repository
	log        *logrus.Entry
}

func NewMarkTaskCompletedUseCase(
	taskRepo task.Repository,
	masterRepo master.Repository,
	log *logrus.Logger,
) *MarkTaskCompletedUseCase {
	return &MarkTaskCompletedUseCase{
		taskRepo:   taskRepo,
		masterRepo: masterRepo,
		log:        log.WithField("usecase", "task.mark_completed"),
	}
}

func (uc *MarkTaskCompletedUseCase) Execute(ctx context.Context, input ChangeTaskStatusInput) (*task.Task, error) {
	log := uc.log.WithFields(logrus.Fields{
		"task_id":  input.TaskID,
		"actor_id": input.ActorID,
	})

	profile, err := uc.masterRepo.GetProfileByUserID(ctx, input.ActorID)
	if err != nil {
		if errors.Is(err, master.ErrProfileNotFound) {
			return nil, task.ErrForbiddenAction
		}
		return nil, fmt.Errorf("get actor master profile: %w", err)
	}

	t, err := uc.taskRepo.GetByID(ctx, input.TaskID)
	if err != nil {
		return nil, err
	}

	if err := t.MarkCompletedByMaster(profile.ID); err != nil {
		log.WithError(err).Warn("mark task completed rejected by domain rules")
		return nil, err
	}

	if err := uc.taskRepo.Update(ctx, t, task.StatusInProgress); err != nil {
		log.WithError(err).Error("failed to update task status to pending_completion")
		return nil, fmt.Errorf("update task: %w", err)
	}

	log.Info("task marked as pending_completion by master")
	return t, nil
}
