package master

import (
	"net/url"
	"regexp"
	"strings"
	"time"

	domaincat "bboard/internal/domain/category"
)

const MaxActiveServices = 3

var phoneRegex = regexp.MustCompile(`^\+7\d{10}$`)

// Profile представляет общий профиль мастера (1 на аккаунт user_id).
type Profile struct {
	ID                   int64
	UserID               int64
	DisplayName          string
	AvatarURL            string
	About                string
	City                 string
	District             string
	ContactPhone         string
	Telegram             string
	IsContactsPublic     bool
	RatingAvg            float64
	ReviewsCount         int
	CompletedOrdersCount int
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// Service представляет отдельную анкету услуги мастера (до 3 активных на профиль).
type Service struct {
	ID                    int64
	MasterProfileID       int64
	Category              string
	Title                 string
	ExperienceDescription string
	Price                 int64
	IsActive              bool
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// ServiceCatalogItem объединяет услугу с краткой информацией профиля мастера для каталога.
type ServiceCatalogItem struct {
	Service
	MasterDisplayName    string
	MasterAvatarURL      string
	MasterCity           string
	MasterDistrict       string
	RatingAvg            float64
	ReviewsCount         int
	CompletedOrdersCount int
}

type profileFields struct {
	displayName      string
	avatarURL        string
	about            string
	city             string
	district         string
	contactPhone     string
	telegram         string
	isContactsPublic bool
}

func normalizeAndValidatePhone(phone string) (string, error) {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return "", ErrEmptyContactPhone
	}

	cleaned := strings.Map(func(r rune) rune {
		if r == ' ' || r == '(' || r == ')' || r == '-' {
			return -1
		}
		return r
	}, phone)

	if strings.HasPrefix(cleaned, "8") && len(cleaned) == 11 {
		cleaned = "+7" + cleaned[1:]
	}

	if !phoneRegex.MatchString(cleaned) {
		return "", ErrInvalidPhone
	}
	return cleaned, nil
}

func validateAndNormalizeProfileFields(
	displayName string,
	avatarURL string,
	about string,
	city string,
	district string,
	contactPhone string,
	telegram string,
	isContactsPublic bool,
) (profileFields, error) {
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		return profileFields{}, ErrEmptyDisplayName
	}
	if len([]rune(displayName)) > 120 {
		return profileFields{}, ErrDisplayNameTooLong
	}

	avatarURL = strings.TrimSpace(avatarURL)
	if avatarURL != "" {
		parsed, err := url.Parse(avatarURL)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			return profileFields{}, ErrInvalidAvatarURL
		}
	}

	normalizedPhone, err := normalizeAndValidatePhone(contactPhone)
	if err != nil {
		return profileFields{}, err
	}

	city = strings.TrimSpace(city)
	if len([]rune(city)) > 100 {
		return profileFields{}, ErrCityTooLong
	}

	district = strings.TrimSpace(district)
	if len([]rune(district)) > 100 {
		return profileFields{}, ErrDistrictTooLong
	}

	telegram = strings.TrimSpace(telegram)
	if len([]rune(telegram)) > 100 {
		return profileFields{}, ErrTelegramTooLong
	}

	return profileFields{
		displayName:      displayName,
		avatarURL:        avatarURL,
		about:            strings.TrimSpace(about),
		city:             city,
		district:         district,
		contactPhone:     normalizedPhone,
		telegram:         telegram,
		isContactsPublic: isContactsPublic,
	}, nil
}

// NewProfile создаёт и валидирует общий профиль мастера.
func NewProfile(
	userID int64,
	displayName string,
	avatarURL string,
	about string,
	city string,
	district string,
	contactPhone string,
	telegram string,
	isContactsPublic bool,
) (*Profile, error) {
	if userID <= 0 {
		return nil, ErrInvalidUserID
	}

	fields, err := validateAndNormalizeProfileFields(
		displayName, avatarURL, about, city, district, contactPhone, telegram, isContactsPublic,
	)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	return &Profile{
		UserID:           userID,
		DisplayName:      fields.displayName,
		AvatarURL:        fields.avatarURL,
		About:            fields.about,
		City:             fields.city,
		District:         fields.district,
		ContactPhone:     fields.contactPhone,
		Telegram:         fields.telegram,
		IsContactsPublic: fields.isContactsPublic,
		CreatedAt:        now,
		UpdatedAt:        now,
	}, nil
}

// Update обновляет общие данные профиля мастера с проверкой прав владельца.
func (p *Profile) Update(
	actorID int64,
	displayName string,
	avatarURL string,
	about string,
	city string,
	district string,
	contactPhone string,
	telegram string,
	isContactsPublic bool,
) error {
	if p.UserID != actorID {
		return ErrForbiddenAction
	}

	fields, err := validateAndNormalizeProfileFields(
		displayName, avatarURL, about, city, district, contactPhone, telegram, isContactsPublic,
	)
	if err != nil {
		return err
	}

	p.DisplayName = fields.displayName
	p.AvatarURL = fields.avatarURL
	p.About = fields.about
	p.City = fields.city
	p.District = fields.district
	p.ContactPhone = fields.contactPhone
	p.Telegram = fields.telegram
	p.IsContactsPublic = fields.isContactsPublic
	p.UpdatedAt = time.Now().UTC()
	return nil
}

type serviceFields struct {
	category              string
	title                 string
	experienceDescription string
	price                 int64
}

func validateAndNormalizeServiceFields(
	category string,
	title string,
	experienceDescription string,
	price int64,
) (serviceFields, error) {
	category = domaincat.Normalize(category)
	if category == "" {
		return serviceFields{}, ErrEmptyCategory
	}
	if !domaincat.IsValid(category) {
		return serviceFields{}, ErrInvalidCategory
	}

	title = strings.TrimSpace(title)
	if title == "" {
		return serviceFields{}, ErrEmptyTitle
	}
	if len([]rune(title)) < 5 {
		return serviceFields{}, ErrTitleTooShort
	}
	if len([]rune(title)) > 200 {
		return serviceFields{}, ErrTitleTooLong
	}

	if price < 0 {
		return serviceFields{}, ErrInvalidPrice
	}

	return serviceFields{
		category:              category,
		title:                 title,
		experienceDescription: strings.TrimSpace(experienceDescription),
		price:                 price,
	}, nil
}

// NewService создаёт и валидирует новую анкету услуги мастера.
func NewService(
	masterProfileID int64,
	category string,
	title string,
	experienceDescription string,
	price int64,
) (*Service, error) {
	if masterProfileID <= 0 {
		return nil, ErrInvalidProfileID
	}

	fields, err := validateAndNormalizeServiceFields(category, title, experienceDescription, price)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	return &Service{
		MasterProfileID:       masterProfileID,
		Category:              fields.category,
		Title:                 fields.title,
		ExperienceDescription: fields.experienceDescription,
		Price:                 fields.price,
		IsActive:              true,
		CreatedAt:             now,
		UpdatedAt:             now,
	}, nil
}

// Update обновляет поля анкеты услуги.
func (s *Service) Update(
	category string,
	title string,
	experienceDescription string,
	price int64,
	isActive bool,
) error {
	fields, err := validateAndNormalizeServiceFields(category, title, experienceDescription, price)
	if err != nil {
		return err
	}

	s.Category = fields.category
	s.Title = fields.title
	s.ExperienceDescription = fields.experienceDescription
	s.Price = fields.price
	s.IsActive = isActive
	s.UpdatedAt = time.Now().UTC()
	return nil
}
