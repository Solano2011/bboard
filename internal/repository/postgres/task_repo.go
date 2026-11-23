package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"bboard/internal/domain/task"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ task.Repository = (*TaskRepo)(nil)

type TaskRepo struct {
	pool *pgxpool.Pool
}

func NewTaskRepo(pool *pgxpool.Pool) *TaskRepo {
	return &TaskRepo{pool: pool}
}

func (r *TaskRepo) Create(ctx context.Context, t *task.Task) error {
	query := `
		INSERT INTO tasks (author_id, master_id, title, description, category, city, budget, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`
	err := r.pool.QueryRow(
		ctx, query,
		t.AuthorID, t.MasterID, t.Title, t.Description,
		t.Category, t.City, t.Budget, t.Status, t.CreatedAt, t.UpdatedAt,
	).Scan(&t.ID)
	if err != nil {
		return fmt.Errorf("insert task: %w", err)
	}
	return nil
}

func (r *TaskRepo) GetByID(ctx context.Context, id int64) (*task.Task, error) {
	query := `
		SELECT id, author_id, master_id, title, description, category, city, budget, status, created_at, updated_at
		FROM tasks
		WHERE id = $1
	`
	var t task.Task
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&t.ID, &t.AuthorID, &t.MasterID, &t.Title, &t.Description,
		&t.Category, &t.City, &t.Budget, &t.Status, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, task.ErrTaskNotFound
		}
		return nil, fmt.Errorf("get task by id: %w", err)
	}
	return &t, nil
}

func (r *TaskRepo) List(ctx context.Context, filter task.ListFilter) ([]*task.Task, error) {
	var conditions []string
	var args []any
	argIdx := 1

	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, filter.Status)
		argIdx++
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
		SELECT id, author_id, master_id, title, description, category, city, budget, status, created_at, updated_at
		FROM tasks
	`
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	var result []*task.Task
	for rows.Next() {
		var t task.Task
		if err := rows.Scan(
			&t.ID, &t.AuthorID, &t.MasterID, &t.Title, &t.Description,
			&t.Category, &t.City, &t.Budget, &t.Status, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan task row: %w", err)
		}
		result = append(result, &t)
	}
	return result, rows.Err()
}

func (r *TaskRepo) Update(ctx context.Context, t *task.Task) error {
	query := `
		UPDATE tasks
		SET master_id = $1, title = $2, description = $3, category = $4,
		    city = $5, budget = $6, status = $7, updated_at = $8
		WHERE id = $9
	`
	cmdTag, err := r.pool.Exec(
		ctx, query,
		t.MasterID, t.Title, t.Description, t.Category,
		t.City, t.Budget, t.Status, t.UpdatedAt, t.ID,
	)
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return task.ErrTaskNotFound
	}
	return nil
}
