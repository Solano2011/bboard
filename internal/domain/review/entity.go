package review

import (
	"strings"
	"time"
)

// Review представляет отзыв заказчика о работе мастера по завершённому заказу.
type Review struct {
	ID              int64
	TaskID          int64
	MasterProfileID int64
	AuthorUserID    int64
	AuthorName      string // Имя автора из users (заполняется при чтении)
	Rating          int
	Comment         string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// NewReview создаёт и валидирует новый отзыв.
func NewReview(taskID, masterProfileID, authorUserID int64, rating int, comment string) (*Review, error) {
	if taskID <= 0 {
		return nil, ErrInvalidTaskID
	}
	if masterProfileID <= 0 {
		return nil, ErrInvalidMasterID
	}
	if authorUserID <= 0 {
		return nil, ErrInvalidAuthorID
	}
	if rating < 1 || rating > 5 {
		return nil, ErrInvalidRating
	}

	now := time.Now().UTC()
	return &Review{
		TaskID:          taskID,
		MasterProfileID: masterProfileID,
		AuthorUserID:    authorUserID,
		Rating:          rating,
		Comment:         strings.TrimSpace(comment),
		CreatedAt:       now,
		UpdatedAt:       now,
	}, nil
}

// Update обновляет оценку и текст отзыва с проверкой прав автора.
func (r *Review) Update(actorUserID int64, rating int, comment string) error {
	if r.AuthorUserID != actorUserID {
		return ErrForbiddenReview
	}
	if rating < 1 || rating > 5 {
		return ErrInvalidRating
	}

	r.Rating = rating
	r.Comment = strings.TrimSpace(comment)
	r.UpdatedAt = time.Now().UTC()
	return nil
}
