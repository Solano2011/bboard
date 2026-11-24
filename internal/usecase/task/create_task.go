package task

import (
	"context"
	"errors"
	"fmt"

	"bboard/internal/domain/task"
	"bboard/internal/domain/user"

	"github.com/sirupsen/logrus"
)

// CreateTaskInput — входные данные для публикации нового объявления заказчика.
type CreateTaskInput struct {
	AuthorID     int64
	Title        string
	Description  string
	Category     string
	City         string
	District     string
	ContactPhone string
	WorkAddress  string
	Budget       int64
}

// CreateTaskUseCase реализует бизнес-сценарий создания нового объявления.
type CreateTaskUseCase struct {
	taskRepo task.Repository
	userRepo user.Repository
	log      *logrus.Entry
}

// NewCreateTaskUseCase собирает зависимости для сценария создания объявления.
func NewCreateTaskUseCase(
	taskRepo task.Repository,
	userRepo user.Repository,
	log *logrus.Logger,
) *CreateTaskUseCase {
	return &CreateTaskUseCase{
		taskRepo: taskRepo,
		userRepo: userRepo,
		log:      log.WithField("usecase", "task.create"),
	}
}

// Execute валидирует входные данные и публикует новое объявление.
func (uc *CreateTaskUseCase) Execute(ctx context.Context, input CreateTaskInput) (*task.Task, error) {
	log := uc.log.WithFields(logrus.Fields{
		"author_id": input.AuthorID,
		"category":  input.Category,
	})

	// 1. Убеждаемся, что автор существует в базе
	if _, err := uc.userRepo.GetByID(ctx, input.AuthorID); err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			log.Warn("create task rejected: author not found")
			return nil, task.ErrInvalidAuthor
		}
		log.WithError(err).Error("failed to verify task author")
		return nil, fmt.Errorf("verify task author: %w", err)
	}

	// 2. Создаём доменную сущность (внутри срабатывают все бизнес-проверки)
	newTask, err := task.NewTask(
		input.AuthorID,
		input.Title,
		input.Description,
		input.Category,
		input.City,
		input.District,
		input.ContactPhone,
		input.WorkAddress,
		input.Budget,
	)
	if err != nil {
		log.WithError(err).Warn("create task rejected: domain validation failed")
		return nil, err
	}

	// 3. Сохраняем объявление через интерфейс репозитория
	if err := uc.taskRepo.Create(ctx, newTask); err != nil {
		log.WithError(err).Error("failed to persist new task")
		return nil, fmt.Errorf("create task in repository: %w", err)
	}

	log.WithField("task_id", newTask.ID).Info("task successfully created")
	return newTask, nil
}
