package task

import (
	"regexp"
	"strings"
	"time"

	domaincat "bboard/internal/domain/category"
)

type Status string

const (
	StatusOpen              Status = "open"               // Открыто, ищем исполнителя
	StatusInProgress        Status = "in_progress"        // Мастер выбран, работа идёт
	StatusPendingCompletion Status = "pending_completion" // Мастер отметил выполненным, ждём подтверждения заказчика
	StatusCompleted         Status = "completed"          // Работа выполнена и подтверждена заказчиком
	StatusCanceled          Status = "canceled"           // Отменено заказчиком
)

var phoneRegex = regexp.MustCompile(`^\+7\d{10}$`)

// Task представляет объявление (заказ) от клиента на выполнение услуги.
type Task struct {
	ID               int64
	AuthorID         int64
	AssignedMasterID *int64 // nil, пока задача открыта; ссылка на master_profiles.id
	Title            string
	Description      string
	Category         string
	City             string
	District         string
	ContactPhone     string
	WorkAddress      string
	Budget           int64
	Status           Status
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type taskFields struct {
	title        string
	description  string
	category     string
	city         string
	district     string
	contactPhone string
	workAddress  string
	budget       int64
}

func validateAndNormalizeFields(
	title string,
	description string,
	category string,
	city string,
	district string,
	contactPhone string,
	workAddress string,
	budget int64,
) (taskFields, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return taskFields{}, ErrEmptyTitle
	}
	if len([]rune(title)) < 5 {
		return taskFields{}, ErrTitleTooShort
	}

	description = strings.TrimSpace(description)
	if description == "" {
		return taskFields{}, ErrEmptyDescription
	}

	category = domaincat.Normalize(category)
	if category == "" {
		return taskFields{}, ErrEmptyCategory
	}
	if !domaincat.IsValid(category) {
		return taskFields{}, ErrInvalidCategory
	}

	district = strings.TrimSpace(district)
	if len([]rune(district)) > 100 {
		return taskFields{}, ErrDistrictTooLong
	}

	contactPhone = strings.TrimSpace(contactPhone)
	if contactPhone != "" {
		cleaned := strings.Map(func(r rune) rune {
			if r == ' ' || r == '(' || r == ')' || r == '-' {
				return -1
			}
			return r
		}, contactPhone)

		if strings.HasPrefix(cleaned, "8") && len(cleaned) == 11 {
			cleaned = "+7" + cleaned[1:]
		}

		if !phoneRegex.MatchString(cleaned) {
			return taskFields{}, ErrInvalidPhone
		}
		contactPhone = cleaned
	}

	workAddress = strings.TrimSpace(workAddress)
	if len([]rune(workAddress)) > 255 {
		return taskFields{}, ErrAddressTooLong
	}

	if budget < 0 {
		return taskFields{}, ErrInvalidBudget
	}

	return taskFields{
		title:        title,
		description:  description,
		category:     category,
		city:         strings.TrimSpace(city),
		district:     district,
		contactPhone: contactPhone,
		workAddress:  workAddress,
		budget:       budget,
	}, nil
}

// NewTask создаёт и валидирует новое объявление заказчика.
func NewTask(
	authorID int64,
	title string,
	description string,
	category string,
	city string,
	district string,
	contactPhone string,
	workAddress string,
	budget int64,
) (*Task, error) {
	if authorID <= 0 {
		return nil, ErrInvalidAuthor
	}

	fields, err := validateAndNormalizeFields(
		title, description, category, city, district, contactPhone, workAddress, budget,
	)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()

	return &Task{
		AuthorID:     authorID,
		Title:        fields.title,
		Description:  fields.description,
		Category:     fields.category,
		City:         fields.city,
		District:     fields.district,
		ContactPhone: fields.contactPhone,
		WorkAddress:  fields.workAddress,
		Budget:       fields.budget,
		Status:       StatusOpen,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

// Update обновляет поля открытого объявления с проверкой прав автора.
func (t *Task) Update(
	actorID int64,
	title string,
	description string,
	category string,
	city string,
	district string,
	contactPhone string,
	workAddress string,
	budget int64,
) error {
	if t.AuthorID != actorID {
		return ErrForbiddenAction
	}
	if t.Status != StatusOpen {
		return ErrInvalidStatus
	}

	fields, err := validateAndNormalizeFields(
		title, description, category, city, district, contactPhone, workAddress, budget,
	)
	if err != nil {
		return err
	}

	t.Title = fields.title
	t.Description = fields.description
	t.Category = fields.category
	t.City = fields.city
	t.District = fields.district
	t.ContactPhone = fields.contactPhone
	t.WorkAddress = fields.workAddress
	t.Budget = fields.budget
	t.UpdatedAt = time.Now().UTC()
	return nil
}

// MarkCompletedByMaster переводит задачу из in_progress в pending_completion назначенным мастером.
func (t *Task) MarkCompletedByMaster(actorProfileID int64) error {
	if t.AssignedMasterID == nil || *t.AssignedMasterID != actorProfileID {
		return ErrForbiddenAction
	}
	if t.Status != StatusInProgress {
		return ErrInvalidStatus
	}

	t.Status = StatusPendingCompletion
	t.UpdatedAt = time.Now().UTC()
	return nil
}

// Complete переводит задачу в статус выполненной. Доступно только автору из in_progress или pending_completion.
func (t *Task) Complete(actorID int64) error {
	if t.AuthorID != actorID {
		return ErrForbiddenAction
	}
	if t.AssignedMasterID == nil {
		return ErrMasterNotAssigned
	}
	if t.Status != StatusInProgress && t.Status != StatusPendingCompletion {
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

// AssignMaster назначает профиль мастера на открытую задачу.
func (t *Task) AssignMaster(actorID, masterProfileID, masterUserID int64) error {
	if t.AuthorID != actorID {
		return ErrForbiddenAction
	}
	if t.Status != StatusOpen {
		return ErrInvalidStatus
	}
	if masterProfileID <= 0 {
		return ErrInvalidMaster
	}
	if masterUserID == t.AuthorID {
		return ErrSelfAssignment
	}

	t.AssignedMasterID = &masterProfileID
	t.Status = StatusInProgress
	t.UpdatedAt = time.Now().UTC()
	return nil
}
