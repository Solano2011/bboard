package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"bboard/internal/domain/master"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ master.Repository = (*MasterRepo)(nil)

type MasterRepo struct {
	pool *pgxpool.Pool
}

func NewMasterRepo(pool *pgxpool.Pool) *MasterRepo {
	return &MasterRepo{pool: pool}
}

func (r *MasterRepo) CreateProfile(ctx context.Context, p *master.Profile) error {
	query := `
		INSERT INTO master_profiles (
			user_id, display_name, avatar_url, about, city, district,
			contact_phone, telegram, is_contacts_public, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id
	`
	err := r.pool.QueryRow(
		ctx, query,
		p.UserID, p.DisplayName, p.AvatarURL, p.About, p.City, p.District,
		p.ContactPhone, p.Telegram, p.IsContactsPublic, p.CreatedAt, p.UpdatedAt,
	).Scan(&p.ID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" && pgErr.ConstraintName == "master_profiles_user_id_key" {
				return master.ErrProfileAlreadyExists
			}
			if pgErr.Code == "23503" {
				return master.ErrInvalidUserID
			}
		}
		return fmt.Errorf("insert master profile: %w", err)
	}
	return nil
}

func (r *MasterRepo) GetProfileByID(ctx context.Context, id int64) (*master.Profile, error) {
	query := `
		SELECT id, user_id, display_name, avatar_url, about, city, district,
		       contact_phone, telegram, is_contacts_public,
		       rating_avg::float8, reviews_count, completed_orders_count,
		       created_at, updated_at
		FROM master_profiles
		WHERE id = $1
	`
	return r.scanProfile(r.pool.QueryRow(ctx, query, id))
}

func (r *MasterRepo) GetProfileByUserID(ctx context.Context, userID int64) (*master.Profile, error) {
	query := `
		SELECT id, user_id, display_name, avatar_url, about, city, district,
		       contact_phone, telegram, is_contacts_public,
		       rating_avg::float8, reviews_count, completed_orders_count,
		       created_at, updated_at
		FROM master_profiles
		WHERE user_id = $1
	`
	return r.scanProfile(r.pool.QueryRow(ctx, query, userID))
}

func (r *MasterRepo) scanProfile(row pgx.Row) (*master.Profile, error) {
	var p master.Profile
	err := row.Scan(
		&p.ID, &p.UserID, &p.DisplayName, &p.AvatarURL, &p.About, &p.City, &p.District,
		&p.ContactPhone, &p.Telegram, &p.IsContactsPublic,
		&p.RatingAvg, &p.ReviewsCount, &p.CompletedOrdersCount,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, master.ErrProfileNotFound
		}
		return nil, fmt.Errorf("scan master profile: %w", err)
	}
	return &p, nil
}

func (r *MasterRepo) UpdateProfile(ctx context.Context, p *master.Profile) error {
	query := `
		UPDATE master_profiles
		SET display_name = $1, avatar_url = $2, about = $3, city = $4, district = $5,
		    contact_phone = $6, telegram = $7, is_contacts_public = $8, updated_at = $9
		WHERE id = $10
	`
	cmdTag, err := r.pool.Exec(
		ctx, query,
		p.DisplayName, p.AvatarURL, p.About, p.City, p.District,
		p.ContactPhone, p.Telegram, p.IsContactsPublic, p.UpdatedAt, p.ID,
	)
	if err != nil {
		return fmt.Errorf("update master profile: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return master.ErrProfileNotFound
	}
	return nil
}

func (r *MasterRepo) DeleteProfile(ctx context.Context, id int64) error {
	cmdTag, err := r.pool.Exec(ctx, `DELETE FROM master_profiles WHERE id = $1`, id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return master.ErrProfileInUse
		}
		return fmt.Errorf("delete master profile: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return master.ErrProfileNotFound
	}
	return nil
}

func (r *MasterRepo) DeleteProfileIfUnused(ctx context.Context, id int64) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin delete profile tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var completedCount, reviewsCount int
	if err := tx.QueryRow(
		ctx,
		`SELECT completed_orders_count, reviews_count FROM master_profiles WHERE id = $1 FOR UPDATE`,
		id,
	).Scan(&completedCount, &reviewsCount); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return master.ErrProfileNotFound
		}
		return fmt.Errorf("lock master profile for delete: %w", err)
	}

	if completedCount > 0 || reviewsCount > 0 {
		return master.ErrProfileInUse
	}

	var hasTasks bool
	if err := tx.QueryRow(
		ctx,
		`SELECT EXISTS (SELECT 1 FROM tasks WHERE assigned_master_id = $1)`,
		id,
	).Scan(&hasTasks); err != nil {
		return fmt.Errorf("check tasks for master profile: %w", err)
	}
	if hasTasks {
		return master.ErrProfileInUse
	}

	var hasReviews bool
	if err := tx.QueryRow(
		ctx,
		`SELECT EXISTS (SELECT 1 FROM master_reviews WHERE master_profile_id = $1)`,
		id,
	).Scan(&hasReviews); err != nil {
		return fmt.Errorf("check reviews for master profile: %w", err)
	}
	if hasReviews {
		return master.ErrProfileInUse
	}

	cmdTag, err := tx.Exec(ctx, `DELETE FROM master_profiles WHERE id = $1`, id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return master.ErrProfileInUse
		}
		return fmt.Errorf("delete master profile: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return master.ErrProfileNotFound
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit delete master profile tx: %w", err)
	}
	return nil
}

func (r *MasterRepo) HasSharedTaskWithProfile(ctx context.Context, viewerUserID, profileID int64) (bool, error) {
	if viewerUserID <= 0 || profileID <= 0 {
		return false, nil
	}
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM tasks
			WHERE author_id = $1
			  AND assigned_master_id = $2
			  AND status IN ('in_progress', 'pending_completion', 'completed')
		)
	`
	var exists bool
	if err := r.pool.QueryRow(ctx, query, viewerUserID, profileID).Scan(&exists); err != nil {
		return false, fmt.Errorf("check shared task with master profile: %w", err)
	}
	return exists, nil
}

func (r *MasterRepo) CreateService(ctx context.Context, s *master.Service) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin create service tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Блокируем строку профиля на время проверки лимита активных анкет
	var lockedProfileID int64
	if err := tx.QueryRow(ctx, `SELECT id FROM master_profiles WHERE id = $1 FOR UPDATE`, s.MasterProfileID).Scan(&lockedProfileID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return master.ErrProfileNotFound
		}
		return fmt.Errorf("lock master profile: %w", err)
	}

	if s.IsActive {
		var activeCount int
		if err := tx.QueryRow(
			ctx,
			`SELECT COUNT(*)::int FROM master_services WHERE master_profile_id = $1 AND is_active = TRUE`,
			s.MasterProfileID,
		).Scan(&activeCount); err != nil {
			return fmt.Errorf("count active services: %w", err)
		}
		if activeCount >= master.MaxActiveServices {
			return master.ErrMaxActiveServicesReached
		}
	}

	query := `
		INSERT INTO master_services (master_profile_id, category, title, experience_description, price, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`
	if err := tx.QueryRow(
		ctx, query,
		s.MasterProfileID, s.Category, s.Title, s.ExperienceDescription, s.Price, s.IsActive, s.CreatedAt, s.UpdatedAt,
	).Scan(&s.ID); err != nil {
		return mapServicePgError(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit create service tx: %w", err)
	}
	return nil
}

func (r *MasterRepo) GetServiceByID(ctx context.Context, id int64) (*master.Service, error) {
	query := `
		SELECT id, master_profile_id, category, title, experience_description, price, is_active, created_at, updated_at
		FROM master_services
		WHERE id = $1
	`
	var s master.Service
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&s.ID, &s.MasterProfileID, &s.Category, &s.Title, &s.ExperienceDescription,
		&s.Price, &s.IsActive, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, master.ErrServiceNotFound
		}
		return nil, fmt.Errorf("get service by id: %w", err)
	}
	return &s, nil
}

func (r *MasterRepo) UpdateService(ctx context.Context, s *master.Service) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin update service tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var lockedProfileID int64
	if err := tx.QueryRow(ctx, `SELECT id FROM master_profiles WHERE id = $1 FOR UPDATE`, s.MasterProfileID).Scan(&lockedProfileID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return master.ErrProfileNotFound
		}
		return fmt.Errorf("lock master profile: %w", err)
	}

	if s.IsActive {
		var activeCount int
		if err := tx.QueryRow(
			ctx,
			`SELECT COUNT(*)::int FROM master_services WHERE master_profile_id = $1 AND is_active = TRUE AND id <> $2`,
			s.MasterProfileID, s.ID,
		).Scan(&activeCount); err != nil {
			return fmt.Errorf("count active services: %w", err)
		}
		if activeCount >= master.MaxActiveServices {
			return master.ErrMaxActiveServicesReached
		}
	}

	query := `
		UPDATE master_services
		SET category = $1, title = $2, experience_description = $3, price = $4, is_active = $5, updated_at = $6
		WHERE id = $7
	`
	cmdTag, err := tx.Exec(
		ctx, query,
		s.Category, s.Title, s.ExperienceDescription, s.Price, s.IsActive, s.UpdatedAt, s.ID,
	)
	if err != nil {
		return mapServicePgError(err)
	}
	if cmdTag.RowsAffected() == 0 {
		return master.ErrServiceNotFound
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit update service tx: %w", err)
	}
	return nil
}

func (r *MasterRepo) DeleteService(ctx context.Context, id int64) error {
	cmdTag, err := r.pool.Exec(ctx, `DELETE FROM master_services WHERE id = $1`, id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return master.ErrProfileInUse
		}
		return fmt.Errorf("delete master service: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return master.ErrServiceNotFound
	}
	return nil
}

func (r *MasterRepo) CountActiveServices(ctx context.Context, profileID int64) (int, error) {
	var count int
	if err := r.pool.QueryRow(
		ctx,
		`SELECT COUNT(*)::int FROM master_services WHERE master_profile_id = $1 AND is_active = TRUE`,
		profileID,
	).Scan(&count); err != nil {
		return 0, fmt.Errorf("count active services: %w", err)
	}
	return count, nil
}

func (r *MasterRepo) ListServicesByProfileID(ctx context.Context, profileID int64, onlyActive bool) ([]*master.Service, error) {
	query := `
		SELECT id, master_profile_id, category, title, experience_description, price, is_active, created_at, updated_at
		FROM master_services
		WHERE master_profile_id = $1
	`
	if onlyActive {
		query += ` AND is_active = TRUE`
	}
	query += ` ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, profileID)
	if err != nil {
		return nil, fmt.Errorf("list services by profile id: %w", err)
	}
	defer rows.Close()

	var result []*master.Service
	for rows.Next() {
		var s master.Service
		if err := rows.Scan(
			&s.ID, &s.MasterProfileID, &s.Category, &s.Title, &s.ExperienceDescription,
			&s.Price, &s.IsActive, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan master service row: %w", err)
		}
		result = append(result, &s)
	}
	return result, rows.Err()
}

func (r *MasterRepo) ListCatalogServices(ctx context.Context, filter master.ServiceListFilter) ([]*master.ServiceCatalogItem, error) {
	var conditions []string
	var args []any
	argIdx := 1

	if filter.OnlyActive {
		conditions = append(conditions, "ms.is_active = TRUE")
	}
	if strings.TrimSpace(filter.Category) != "" {
		conditions = append(conditions, fmt.Sprintf("ms.category = $%d", argIdx))
		args = append(args, strings.ToLower(strings.TrimSpace(filter.Category)))
		argIdx++
	}
	if strings.TrimSpace(filter.City) != "" {
		conditions = append(conditions, fmt.Sprintf("LOWER(mp.city) = LOWER($%d)", argIdx))
		args = append(args, strings.TrimSpace(filter.City))
		argIdx++
	}

	query := `
		SELECT ms.id, ms.master_profile_id, ms.category, ms.title, ms.experience_description,
		       ms.price, ms.is_active, ms.created_at, ms.updated_at,
		       mp.display_name, mp.avatar_url, mp.city, mp.district,
		       mp.rating_avg::float8, mp.reviews_count, mp.completed_orders_count
		FROM master_services ms
		JOIN master_profiles mp ON mp.id = ms.master_profile_id
	`
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += fmt.Sprintf(" ORDER BY mp.rating_avg DESC, mp.completed_orders_count DESC, ms.created_at DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list catalog services: %w", err)
	}
	defer rows.Close()

	var result []*master.ServiceCatalogItem
	for rows.Next() {
		var item master.ServiceCatalogItem
		if err := rows.Scan(
			&item.ID, &item.MasterProfileID, &item.Category, &item.Title, &item.ExperienceDescription,
			&item.Price, &item.IsActive, &item.CreatedAt, &item.UpdatedAt,
			&item.MasterDisplayName, &item.MasterAvatarURL, &item.MasterCity, &item.MasterDistrict,
			&item.RatingAvg, &item.ReviewsCount, &item.CompletedOrdersCount,
		); err != nil {
			return nil, fmt.Errorf("scan catalog service row: %w", err)
		}
		result = append(result, &item)
	}
	return result, rows.Err()
}

func mapServicePgError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			if pgErr.ConstraintName == "uq_master_services_active_category" {
				return master.ErrServiceCategoryExists
			}
		case "23503":
			return master.ErrProfileNotFound
		case "23514":
			return master.ErrInvalidPrice
		}
	}
	return fmt.Errorf("persist master service: %w", err)
}
