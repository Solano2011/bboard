package bid

import (
	"strings"
	"time"
)

// Bid представляет отклик мастера на объявление заказчика.
type Bid struct {
	ID            int64
	TaskID        int64
	MasterID      int64
	ProposedPrice int64  // Предложенная мастером цена в рублях
	Comment       string // Сопроводительный текст от мастера
	CreatedAt     time.Time
}

// NewBid создаёт и валидирует новый отклик мастера.
func NewBid(taskID, masterID, proposedPrice int64, comment string) (*Bid, error) {
	if taskID <= 0 {
		return nil, ErrInvalidTaskID
	}
	if masterID <= 0 {
		return nil, ErrInvalidMasterID
	}
	if proposedPrice < 0 {
		return nil, ErrInvalidPrice
	}

	comment = strings.TrimSpace(comment)
	if comment == "" {
		return nil, ErrEmptyComment
	}
	if len([]rune(comment)) < 5 {
		return nil, ErrCommentTooShort
	}

	return &Bid{
		TaskID:        taskID,
		MasterID:      masterID,
		ProposedPrice: proposedPrice,
		Comment:       comment,
		CreatedAt:     time.Now().UTC(),
	}, nil
}
