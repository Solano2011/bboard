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
		INSERT INTO task_bids (task_id, master_id, proposed_price, comment, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`
	err := r.pool.QueryRow(
		ctx, query,
		b.TaskID, b.MasterID, b.ProposedPrice, b.Comment, b.CreatedAt,
	).Scan(&b.ID)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return bid.ErrBidAlreadyExists
		}
		return fmt.Errorf("insert task bid: %w", err)
	}
	return nil
}

func (r *BidRepo) ListByTaskID(ctx context.Context, taskID int64) ([]*bid.Bid, error) {
	query := `
		SELECT id, task_id, master_id, proposed_price, comment, created_at
		FROM task_bids
		WHERE task_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, query, taskID)
	if err != nil {
		return nil, fmt.Errorf("list task bids: %w", err)
	}
	defer rows.Close()

	var result []*bid.Bid
	for rows.Next() {
		var b bid.Bid
		if err := rows.Scan(
			&b.ID, &b.TaskID, &b.MasterID, &b.ProposedPrice, &b.Comment, &b.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan task bid row: %w", err)
		}
		result = append(result, &b)
	}
	return result, rows.Err()
}
