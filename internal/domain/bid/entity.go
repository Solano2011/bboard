package bid

import (
	"strings"
	"time"
)

// Bid представляет отклик мастера (task_responses) на объявление заказчика.
type Bid struct {
	ID              int64
	TaskID          int64
	MasterProfileID int64
	ProposedPrice   *int64 // Опционально предложенная мастером цена в рублях
	Comment         string // Опциональный сопроводительный текст от мастера
	CreatedAt       time.Time

	// Краткая информация о профиле мастера для списка откликов (заполняется при чтении)
	MasterDisplayName          string
	MasterAvatarURL            string
	MasterRatingAvg            float64
	MasterReviewsCount         int
	MasterCompletedOrdersCount int
}

// NewBid создаёт и валидирует новый отклик мастера.
func NewBid(taskID, masterProfileID int64, proposedPrice *int64, comment string) (*Bid, error) {
	if taskID <= 0 {
		return nil, ErrInvalidTaskID
	}
	if masterProfileID <= 0 {
		return nil, ErrInvalidMasterID
	}
	if proposedPrice != nil && *proposedPrice < 0 {
		return nil, ErrInvalidPrice
	}

	return &Bid{
		TaskID:          taskID,
		MasterProfileID: masterProfileID,
		ProposedPrice:   proposedPrice,
		Comment:         strings.TrimSpace(comment),
		CreatedAt:       time.Now().UTC(),
	}, nil
}
