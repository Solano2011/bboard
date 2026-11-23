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

func (r *MasterRepo) Create(ctx context.Context, p *master.Profile) error {
	query := `
		INSERT INTO master_profiles (user_id, headline, bio, category, city, experience_years, min_price, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`
	err := r.pool.QueryRow(
		ctx, query,
		p.UserID, p.Headline, p.Bio, p.Category, p.City,
		p.ExperienceYears, p.MinPrice, p.IsActive, p.CreatedAt, p.UpdatedAt,
	).Scan(&p.ID)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return master.ErrProfileAlreadyExists
		}
		return fmt.Errorf("insert master profile: %w", err)
	}
	return nil
}

func (r *MasterRepo) GetByID(ctx context.Context, id int64) (*master.Profile, error) {
	query := `
		SELECT id, user_id, headline, bio, category, city, experience_years, min_price, is_active, created_at, updated_at
		FROM master_profiles
		WHERE id = $1
	`
	return r.scanOne(ctx, query, id)
}

func (r *MasterRepo) GetByUserID(ctx context.Context, userID int64) (*master.Profile, error) {
	query := `
		SELECT id, user_id, headline, bio, category, city, experience_years, min_price, is_active, created_at, updated_at
		FROM master_profiles
		WHERE user_id = $1
	`
	return r.scanOne(ctx, query, userID)
}

func (r *MasterRepo) scanOne(ctx context.Context, query string, arg any) (*master.Profile, error) {
	var p master.Profile
	err := r.pool.QueryRow(ctx, query, arg).Scan(
		&p.ID, &p.UserID, &p.Headline, &p.Bio, &p.Category,
		&p.City, &p.ExperienceYears, &p.MinPrice, &p.IsActive, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, master.ErrProfileNotFound
		}
		return nil, fmt.Errorf("select master profile: %w", err)
	}
	return &p, nil
}

func (r *MasterRepo) List(ctx context.Context, filter master.ListFilter) ([]*master.Profile, error) {
	var conditions []string
	var args []any
	argIdx := 1

	if filter.OnlyActive {
		conditions = append(conditions, "is_active = TRUE")
	}
	if strings.TrimSpace(filter.Category) != "" {
		conditions = append(conditions, fmt.Sprintf("category = $%d", argIdx))
		args = append(args, strings.ToLower(strings.TrimSpace(filter.Category)))
		argIdx++
	}
	if strings.TrimSpace(filter.City) != "" {
		conditions = append(conditions, fmt.Sprintf("LOWER(city) = LOWER($%d)", argIdx))
		args = append(args, strings.TrimSpace(filter.City))
		argIdx++
	}

	query := `
		SELECT id, user_id, headline, bio, category, city, experience_years, min_price, is_active, created_at, updated_at
		FROM master_profiles
	`
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += fmt.Sprintf(" ORDER BY experience_years DESC, created_at DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list master profiles: %w", err)
	}
	defer rows.Close()

	var result []*master.Profile
	for rows.Next() {
		var p master.Profile
		if err := rows.Scan(
			&p.ID, &p.UserID, &p.Headline, &p.Bio, &p.Category,
			&p.City, &p.ExperienceYears, &p.MinPrice, &p.IsActive, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan master profile row: %w", err)
		}
		result = append(result, &p)
	}
	return result, rows.Err()
}

func (r *MasterRepo) Update(ctx context.Context, p *master.Profile) error {
	query := `
		UPDATE master_profiles
		SET headline = $1, bio = $2, category = $3, city = $4,
		    experience_years = $5, min_price = $6, is_active = $7, updated_at = $8
		WHERE id = $9
	`
	cmdTag, err := r.pool.Exec(
		ctx, query,
		p.Headline, p.Bio, p.Category, p.City,
		p.ExperienceYears, p.MinPrice, p.IsActive, p.UpdatedAt, p.ID,
	)
	if err != nil {
		return fmt.Errorf("update master profile: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return master.ErrProfileNotFound
	}
	return nil
}
