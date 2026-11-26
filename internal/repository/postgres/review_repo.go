package postgres

import (
	"context"
	"errors"
	"fmt"

	"bboard/internal/domain/review"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ review.Repository = (*ReviewRepo)(nil)

type ReviewRepo struct {
	pool *pgxpool.Pool
}

func NewReviewRepo(pool *pgxpool.Pool) *ReviewRepo {
	return &ReviewRepo{pool: pool}
}

func lockMasterProfileTx(ctx context.Context, tx pgx.Tx, masterProfileID int64) error {
	var lockedID int64
	if err := tx.QueryRow(
		ctx,
		`SELECT id FROM master_profiles WHERE id = $1 FOR UPDATE`,
		masterProfileID,
	).Scan(&lockedID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return review.ErrInvalidMasterID
		}
		return fmt.Errorf("lock master profile for review: %w", err)
	}
	return nil
}

func recalculateMasterRatingTx(ctx context.Context, tx pgx.Tx, masterProfileID int64) error {
	query := `
		UPDATE master_profiles
		SET rating_avg = COALESCE((
				SELECT ROUND(AVG(rating)::numeric, 2)
				FROM master_reviews
				WHERE master_profile_id = $1
			), 0),
		    reviews_count = (
				SELECT COUNT(*)::int
				FROM master_reviews
				WHERE master_profile_id = $1
			),
		    updated_at = NOW()
		WHERE id = $1
	`
	cmdTag, err := tx.Exec(ctx, query, masterProfileID)
	if err != nil {
		return fmt.Errorf("recalculate master rating: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return review.ErrInvalidMasterID
	}
	return nil
}

func (r *ReviewRepo) Create(ctx context.Context, rev *review.Review) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin create review tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := lockMasterProfileTx(ctx, tx, rev.MasterProfileID); err != nil {
		return err
	}

	query := `
		INSERT INTO master_reviews (task_id, master_profile_id, author_user_id, rating, comment, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`
	if err := tx.QueryRow(
		ctx, query,
		rev.TaskID, rev.MasterProfileID, rev.AuthorUserID, rev.Rating, rev.Comment, rev.CreatedAt, rev.UpdatedAt,
	).Scan(&rev.ID); err != nil {
		return mapReviewPgError(err, "insert review")
	}

	if err := recalculateMasterRatingTx(ctx, tx, rev.MasterProfileID); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit create review tx: %w", err)
	}
	return nil
}

func (r *ReviewRepo) GetByID(ctx context.Context, id int64) (*review.Review, error) {
	query := `
		SELECT mr.id, mr.task_id, mr.master_profile_id, mr.author_user_id, u.name,
		       mr.rating, mr.comment, mr.created_at, mr.updated_at
		FROM master_reviews mr
		JOIN users u ON u.id = mr.author_user_id
		WHERE mr.id = $1
	`
	return r.scanReview(r.pool.QueryRow(ctx, query, id))
}

func (r *ReviewRepo) GetByTaskID(ctx context.Context, taskID int64) (*review.Review, error) {
	query := `
		SELECT mr.id, mr.task_id, mr.master_profile_id, mr.author_user_id, u.name,
		       mr.rating, mr.comment, mr.created_at, mr.updated_at
		FROM master_reviews mr
		JOIN users u ON u.id = mr.author_user_id
		WHERE mr.task_id = $1
	`
	return r.scanReview(r.pool.QueryRow(ctx, query, taskID))
}

func (r *ReviewRepo) GetByTaskIDs(ctx context.Context, taskIDs []int64) (map[int64]*review.Review, error) {
	result := make(map[int64]*review.Review)
	if len(taskIDs) == 0 {
		return result, nil
	}

	query := `
		SELECT mr.id, mr.task_id, mr.master_profile_id, mr.author_user_id, u.name,
		       mr.rating, mr.comment, mr.created_at, mr.updated_at
		FROM master_reviews mr
		JOIN users u ON u.id = mr.author_user_id
		WHERE mr.task_id = ANY($1)
	`
	rows, err := r.pool.Query(ctx, query, taskIDs)
	if err != nil {
		return nil, fmt.Errorf("get reviews by task ids: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var rev review.Review
		if err := rows.Scan(
			&rev.ID, &rev.TaskID, &rev.MasterProfileID, &rev.AuthorUserID, &rev.AuthorName,
			&rev.Rating, &rev.Comment, &rev.CreatedAt, &rev.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan review row: %w", err)
		}
		result[rev.TaskID] = &rev
	}
	return result, rows.Err()
}

func (r *ReviewRepo) scanReview(row pgx.Row) (*review.Review, error) {
	var rev review.Review
	err := row.Scan(
		&rev.ID, &rev.TaskID, &rev.MasterProfileID, &rev.AuthorUserID, &rev.AuthorName,
		&rev.Rating, &rev.Comment, &rev.CreatedAt, &rev.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, review.ErrReviewNotFound
		}
		return nil, fmt.Errorf("scan review: %w", err)
	}
	return &rev, nil
}

func (r *ReviewRepo) ListByMasterProfileID(ctx context.Context, masterProfileID int64, limit, offset int) ([]*review.Review, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT mr.id, mr.task_id, mr.master_profile_id, mr.author_user_id, u.name,
		       mr.rating, mr.comment, mr.created_at, mr.updated_at
		FROM master_reviews mr
		JOIN users u ON u.id = mr.author_user_id
		WHERE mr.master_profile_id = $1
		ORDER BY mr.created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.pool.Query(ctx, query, masterProfileID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list reviews by master profile id: %w", err)
	}
	defer rows.Close()

	var result []*review.Review
	for rows.Next() {
		var rev review.Review
		if err := rows.Scan(
			&rev.ID, &rev.TaskID, &rev.MasterProfileID, &rev.AuthorUserID, &rev.AuthorName,
			&rev.Rating, &rev.Comment, &rev.CreatedAt, &rev.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan review row: %w", err)
		}
		result = append(result, &rev)
	}
	return result, rows.Err()
}

func (r *ReviewRepo) Update(ctx context.Context, rev *review.Review) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin update review tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := lockMasterProfileTx(ctx, tx, rev.MasterProfileID); err != nil {
		return err
	}

	cmdTag, err := tx.Exec(
		ctx,
		`UPDATE master_reviews SET rating = $1, comment = $2, updated_at = $3 WHERE id = $4`,
		rev.Rating, rev.Comment, rev.UpdatedAt, rev.ID,
	)
	if err != nil {
		return mapReviewPgError(err, "update review")
	}
	if cmdTag.RowsAffected() == 0 {
		return review.ErrReviewNotFound
	}

	if err := recalculateMasterRatingTx(ctx, tx, rev.MasterProfileID); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit update review tx: %w", err)
	}
	return nil
}

func (r *ReviewRepo) Delete(ctx context.Context, id int64) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin delete review tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var masterProfileID int64
	if err := tx.QueryRow(
		ctx,
		`SELECT master_profile_id FROM master_reviews WHERE id = $1`,
		id,
	).Scan(&masterProfileID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return review.ErrReviewNotFound
		}
		return fmt.Errorf("find review for delete: %w", err)
	}

	if err := lockMasterProfileTx(ctx, tx, masterProfileID); err != nil {
		return err
	}

	cmdTag, err := tx.Exec(ctx, `DELETE FROM master_reviews WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete review: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return review.ErrReviewNotFound
	}

	if err := recalculateMasterRatingTx(ctx, tx, masterProfileID); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit delete review tx: %w", err)
	}
	return nil
}

func mapReviewPgError(err error, action string) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			if pgErr.ConstraintName == "master_reviews_task_id_key" {
				return review.ErrReviewAlreadyExists
			}
		case "23503":
			return review.ErrInvalidTaskID
		case "23514":
			return review.ErrInvalidRating
		}
	}
	return fmt.Errorf("%s: %w", action, err)
}
