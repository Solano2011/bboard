package task

import (
	"strings"
	"time"
)

type Status string

const (
	StatusOpen       Status = "open"        // Открыто, ищем исполнителя
	StatusInProgress Status = "in_progress" // Мастер выбран, работа идёт
	StatusCompleted  Status = "completed"   // Работа выполнена
	StatusCanceled   Status = "canceled"    // Отменено заказчиком
)

// Task представляет объявление (заказ) от клиента на выполнение услуги.
type Task struct {
	ID          int64
	AuthorID    int64
	MasterID    *int64 // nil, пока задача открыта; заполняется при переходе в in_progress
	Title       string
	Description string
	Category    string
	City        string
	Budget      int64
	Status      Status
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NewTask создаёт и валидирует новое объявление заказчика.
func NewTask(
	authorID int64,
	title string,
	description string,
	category string,
	city string,
	budget int64,
) (*Task, error) {
	if authorID <= 0 {
		return nil, ErrInvalidAuthor
	}

	title = strings.TrimSpace(title)
	if title == "" {
		return nil, ErrEmptyTitle
	}
	if len([]rune(title)) < 5 {
		return nil, ErrTitleTooShort
	}

	description = strings.TrimSpace(description)
	if description == "" {
		return nil, ErrEmptyDescription
	}

	category = strings.ToLower(strings.TrimSpace(category))
	if category == "" {
		return nil, ErrEmptyCategory
	}

	if budget < 0 {
		return nil, ErrInvalidBudget
	}

	now := time.Now().UTC()

	return &Task{
		AuthorID:    authorID,
		Title:       title,
		Description: description,
		Category:    category,
		City:        strings.TrimSpace(city),
		Budget:      budget,
		Status:      StatusOpen,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// Complete переводит задачу в статус выполненной. Доступно только автору открытой/активной задачи.
func (t *Task) Complete(actorID int64) error {
	if t.AuthorID != actorID {
		return ErrForbiddenAction
	}
	if t.Status != StatusInProgress || t.MasterID == nil {
		return ErrInvalidStatus
	}

	t.Status = StatusCompleted
	t.UpdatedAt = time.Now().UTC()
	return nil
}

// Cancel отменяет объявление.
func (t *Task) Cancel(actorID int64) error {
	if t.AuthorID != actorID {
		return ErrForbiddenAction
	}
	if t.Status == StatusCompleted || t.Status == StatusCanceled {
		return ErrInvalidStatus
	}

	t.Status = StatusCanceled
	t.UpdatedAt = time.Now().UTC()
	return nil
}

func (t *Task) AssignMaster(actorID, masterUserID int64) error {
	// 1. Назначить исполнителя может только автор объявления
	if t.AuthorID != actorID {
		return ErrForbiddenAction
	}
	// 2. Задача должна быть открыта
	if t.Status != StatusOpen {
		return ErrInvalidStatus
	}
	// 3. Нельзя назначить самого себя исполнителем своего же заказа
	if masterUserID <= 0 {
		return ErrInvalidMaster
	}
	if masterUserID == t.AuthorID {
		return ErrSelfAssignment
	}

	t.MasterID = &masterUserID
	t.Status = StatusInProgress
	t.UpdatedAt = time.Now().UTC()
	return nil
}
