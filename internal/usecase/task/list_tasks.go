package task

import (
	"context"
	"fmt"

	"bboard/internal/domain/task"

	"github.com/sirupsen/logrus"
)

type ListTasksInput struct {
	Category string
	City     string
	Status   task.Status
	Limit    int
	Offset   int
}

type ListTasksUseCase struct {
	taskRepo task.Repository
	log      *logrus.Entry
}

func NewListTasksUseCase(taskRepo task.Repository, log *logrus.Logger) *ListTasksUseCase {
	return &ListTasksUseCase{
		taskRepo: taskRepo,
		log:      log.WithField("usecase", "task.list"),
	}
}

func (uc *ListTasksUseCase) Execute(ctx context.Context, input ListTasksInput) ([]*task.Task, error) {
	if input.Limit <= 0 || input.Limit > 100 {
		input.Limit = 20
	}
	if input.Offset < 0 {
		input.Offset = 0
	}
	if input.Status == "" {
		input.Status = task.StatusOpen
	}

	tasks, err := uc.taskRepo.List(ctx, task.ListFilter{
		Category: input.Category,
		City:     input.City,
		Status:   input.Status,
		Limit:    input.Limit,
		Offset:   input.Offset,
	})
	if err != nil {
		uc.log.WithError(err).Error("failed to list tasks")
		return nil, fmt.Errorf("list tasks: %w", err)
	}

	return tasks, nil
}
