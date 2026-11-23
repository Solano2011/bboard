package task

import (
	"context"
	"errors"
	"fmt"

	"bboard/internal/domain/master"
	"bboard/internal/domain/task"

	"github.com/sirupsen/logrus"
)

type AssignMasterInput struct {
	TaskID   int64
	ActorID  int64
	MasterID int64
}

type AssignMasterUseCase struct {
	taskRepo   task.Repository
	masterRepo master.Repository
	log        *logrus.Entry
}

func NewAssignMasterUseCase(
	taskRepo task.Repository,
	masterRepo master.Repository,
	log *logrus.Logger,
) *AssignMasterUseCase {
	return &AssignMasterUseCase{
		taskRepo:   taskRepo,
		masterRepo: masterRepo,
		log:        log.WithField("usecase", "task.assign_master"),
	}
}

func (uc *AssignMasterUseCase) Execute(ctx context.Context, input AssignMasterInput) (*task.Task, error) {
	log := uc.log.WithFields(logrus.Fields{
		"task_id":   input.TaskID,
		"actor_id":  input.ActorID,
		"master_id": input.MasterID,
	})

	// 1. Проверяем, что назначаемый пользователь зарегистрирован как мастер и его анкета активна
	profile, err := uc.masterRepo.GetByUserID(ctx, input.MasterID)
	if err != nil {
		if errors.Is(err, master.ErrProfileNotFound) {
			log.Warn("assign master rejected: user has no master profile")
			return nil, task.ErrInvalidMaster
		}
		return nil, fmt.Errorf("verify master profile: %w", err)
	}
	if !profile.IsActive {
		log.Warn("assign master rejected: master profile is inactive")
		return nil, task.ErrInvalidMaster
	}

	// 2. Загружаем задачу и выполняем доменный переход в in_progress
	t, err := uc.taskRepo.GetByID(ctx, input.TaskID)
	if err != nil {
		return nil, err
	}

	if err := t.AssignMaster(input.ActorID, input.MasterID); err != nil {
		log.WithError(err).Warn("assign master rejected by domain rules")
		return nil, err
	}

	if err := uc.taskRepo.Update(ctx, t); err != nil {
		log.WithError(err).Error("failed to update task after master assignment")
		return nil, fmt.Errorf("update task: %w", err)
	}

	log.Info("master assigned to task")
	return t, nil
}
