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
	TaskID          int64
	ActorID         int64
	MasterProfileID int64
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
		"task_id":           input.TaskID,
		"actor_id":          input.ActorID,
		"master_profile_id": input.MasterProfileID,
	})

	if input.MasterProfileID <= 0 {
		return nil, task.ErrInvalidMaster
	}

	profile, err := uc.masterRepo.GetProfileByID(ctx, input.MasterProfileID)
	if err != nil {
		if errors.Is(err, master.ErrProfileNotFound) {
			log.Warn("assign master rejected: master profile not found")
			return nil, task.ErrInvalidMaster
		}
		return nil, fmt.Errorf("verify master profile: %w", err)
	}

	t, err := uc.taskRepo.GetByID(ctx, input.TaskID)
	if err != nil {
		return nil, err
	}

	if err := t.AssignMaster(input.ActorID, profile.ID, profile.UserID); err != nil {
		log.WithError(err).Warn("assign master rejected by domain rules")
		return nil, err
	}

	if err := uc.taskRepo.Update(ctx, t, task.StatusOpen); err != nil {
		log.WithError(err).Error("failed to update task after master assignment")
		return nil, fmt.Errorf("update task: %w", err)
	}

	log.Info("master assigned to task")
	return t, nil
}
