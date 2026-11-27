package task

import (
	"context"

	"bboard/internal/domain/task"
	"bboard/internal/domain/user"

	"github.com/sirupsen/logrus"
)

type DeleteTaskPhotoInput struct {
	TaskID    int64
	PhotoID   int64
	ActorID   int64
	ActorRole user.Role
}

type DeleteTaskPhotoUseCase struct {
	taskRepo     task.Repository
	photoStorage PhotoStorage
	log          *logrus.Entry
}

func NewDeleteTaskPhotoUseCase(
	taskRepo task.Repository,
	photoStorage PhotoStorage,
	log *logrus.Logger,
) *DeleteTaskPhotoUseCase {
	return &DeleteTaskPhotoUseCase{
		taskRepo:     taskRepo,
		photoStorage: photoStorage,
		log:          log.WithField("usecase", "task.delete_photo"),
	}
}

func (uc *DeleteTaskPhotoUseCase) Execute(ctx context.Context, input DeleteTaskPhotoInput) error {
	if input.TaskID <= 0 {
		return task.ErrTaskNotFound
	}
	if input.PhotoID <= 0 {
		return task.ErrPhotoNotFound
	}

	t, err := uc.taskRepo.GetByID(ctx, input.TaskID)
	if err != nil {
		return err
	}
	if err := t.CanManagePhotos(input.ActorID, input.ActorRole); err != nil {
		return err
	}

	expectedStatuses := []task.Status{task.StatusOpen}
	if input.ActorRole.IsModeratorOrAdmin() {
		expectedStatuses = []task.Status{
			task.StatusOpen,
			task.StatusInProgress,
			task.StatusPendingCompletion,
		}
	}

	deleted, err := uc.taskRepo.DeletePhoto(ctx, input.TaskID, input.PhotoID, expectedStatuses...)
	if err != nil {
		return err
	}

	if err := uc.photoStorage.Delete(ctx, deleted.StorageKey); err != nil {
		uc.log.WithError(err).WithField("storage_key", deleted.StorageKey).Warn("failed to delete photo file from disk")
	}

	return nil
}
