package postgres

import (
	"context"
	"errors"
	"fmt"

	"bboard/internal/domain/bid"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ bid.Repository = (*BidRepo)(nil)

type BidRepo struct {
	pool *pgxpool.Pool
}

func NewBidRepo(pool *pgxpool.Pool) *BidRepo {
	return &BidRepo{pool: pool}
}

func (r *BidRepo) Create(ctx context.Context, b *bid.Bid) error {
	query := `
		INSERT INTO task_responses (task_id, master_profile_id, proposed_price, comment, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`
	err := r.pool.QueryRow(
		ctx, query,
		b.TaskID, b.MasterProfileID, b.ProposedPrice, b.Comment, b.CreatedAt,
	).Scan(&b.ID)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "23505":
				if pgErr.ConstraintName == "uq_task_responses_task_master" {
					return bid.ErrBidAlreadyExists
				}
			case "23503":
				return bid.ErrInvalidTaskID
			case "23514":
				return bid.ErrInvalidPrice
			}
		}
		return fmt.Errorf("insert task response: %w", err)
	}
	return nil
}

func (r *BidRepo) ListByTaskID(ctx context.Context, taskID int64) ([]*bid.Bid, error) {
	query := `
		SELECT tr.id, tr.task_id, tr.master_profile_id, tr.proposed_price, tr.comment, tr.created_at,
		       mp.display_name, mp.avatar_url, mp.rating_avg::float8, mp.reviews_count, mp.completed_orders_count
		FROM task_responses tr
		JOIN master_profiles mp ON mp.id = tr.master_profile_id
		WHERE tr.task_id = $1
		ORDER BY tr.created_at DESC
	`
	rows, err := r.pool.Query(ctx, query, taskID)
	if err != nil {
		return nil, fmt.Errorf("list task responses: %w", err)
	}
	defer rows.Close()

	var result []*bid.Bid
	for rows.Next() {
		var b bid.Bid
		if err := rows.Scan(
			&b.ID, &b.TaskID, &b.MasterProfileID, &b.ProposedPrice, &b.Comment, &b.CreatedAt,
			&b.MasterDisplayName, &b.MasterAvatarURL, &b.MasterRatingAvg, &b.MasterReviewsCount, &b.MasterCompletedOrdersCount,
		); err != nil {
			return nil, fmt.Errorf("scan task response row: %w", err)
		}
		result = append(result, &b)
	}
	return result, rows.Err()
}
