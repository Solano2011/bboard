package master

import (
	"strings"
	"time"
)

// Profile представляет анкету исполнителя (мастера) на доске объявлений.
type Profile struct {
	ID              int64
	UserID          int64
	Headline        string // Например: "Электромонтажные работы любой сложности"
	Bio             string // Подробное описание услуг, инструмента, условий
	Category        string // Основная специализация: "electric", "plumbing", "earthworks", "repair"
	City            string
	ExperienceYears int
	MinPrice        int64 // Минимальная стоимость выезда/работы в рублях
	IsActive        bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// NewProfile создаёт и валидирует новую анкету мастера.
func NewProfile(
	userID int64,
	headline string,
	bio string,
	category string,
	city string,
	experienceYears int,
	minPrice int64,
) (*Profile, error) {
	if userID <= 0 {
		return nil, ErrInvalidUserID
	}

	headline = strings.TrimSpace(headline)
	if headline == "" {
		return nil, ErrEmptyHeadline
	}
	if len([]rune(headline)) < 5 {
		return nil, ErrHeadlineTooShort
	}

	bio = strings.TrimSpace(bio)
	if bio == "" {
		return nil, ErrEmptyBio
	}

	category = strings.ToLower(strings.TrimSpace(category))
	if category == "" {
		return nil, ErrEmptyCategory
	}

	if experienceYears < 0 {
		return nil, ErrInvalidExperience
	}

	if minPrice < 0 {
		return nil, ErrInvalidMinPrice
	}

	now := time.Now().UTC()

	return &Profile{
		UserID:          userID,
		Headline:        headline,
		Bio:             bio,
		Category:        category,
		City:            strings.TrimSpace(city),
		ExperienceYears: experienceYears,
		MinPrice:        minPrice,
		IsActive:        true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}, nil
}

// Update обновляет данные анкеты мастера с проверкой прав владельца.
func (p *Profile) Update(
	actorID int64,
	headline string,
	bio string,
	category string,
	city string,
	experienceYears int,
	minPrice int64,
	isActive bool,
) error {
	if p.UserID != actorID {
		return ErrForbiddenAction
	}

	headline = strings.TrimSpace(headline)
	if headline == "" {
		return ErrEmptyHeadline
	}
	if len([]rune(headline)) < 5 {
		return ErrHeadlineTooShort
	}

	bio = strings.TrimSpace(bio)
	if bio == "" {
		return ErrEmptyBio
	}

	category = strings.ToLower(strings.TrimSpace(category))
	if category == "" {
		return ErrEmptyCategory
	}

	if experienceYears < 0 {
		return ErrInvalidExperience
	}
	if minPrice < 0 {
		return ErrInvalidMinPrice
	}

	p.Headline = headline
	p.Bio = bio
	p.Category = category
	p.City = strings.TrimSpace(city)
	p.ExperienceYears = experienceYears
	p.MinPrice = minPrice
	p.IsActive = isActive
	p.UpdatedAt = time.Now().UTC()
	return nil
}
