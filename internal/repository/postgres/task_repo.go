package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"bboard/internal/domain/task"
	"bboard/internal/domain/user"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
		INSERT INTO tasks (author_id, assigned_master_id, title, description, category, city, district, contact_phone, work_address, budget, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id
	`
	err := r.pool.QueryRow(
		ctx, query,
		t.AuthorID, t.AssignedMasterID, t.Title, t.Description,
		t.Category, t.City, t.District, t.ContactPhone, t.WorkAddress,
		t.Budget, t.Status, t.CreatedAt, t.UpdatedAt,
	).Scan(&t.ID)
	if err != nil {
		return mapTaskPgError(err, "insert task")
	}
	return nil
}

func (r *TaskRepo) GetByID(ctx context.Context, id int64) (*task.Task, error) {
	query := `
		SELECT id, author_id, assigned_master_id, title, description, category, city, district, contact_phone, work_address, budget, status, created_at, updated_at
		FROM tasks
		WHERE id = $1
	`
	var t task.Task
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&t.ID, &t.AuthorID, &t.AssignedMasterID, &t.Title, &t.Description,
		&t.Category, &t.City, &t.District, &t.ContactPhone, &t.WorkAddress,
		&t.Budget, &t.Status, &t.CreatedAt, &t.UpdatedAt,
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
		SELECT id, author_id, assigned_master_id, title, description, category, city, district, contact_phone, work_address, budget, status, created_at, updated_at
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
			&t.ID, &t.AuthorID, &t.AssignedMasterID, &t.Title, &t.Description,
			&t.Category, &t.City, &t.District, &t.ContactPhone, &t.WorkAddress,
			&t.Budget, &t.Status, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan task row: %w", err)
		}
		result = append(result, &t)
	}
	return result, rows.Err()
}

func (r *TaskRepo) Update(ctx context.Context, t *task.Task, allowedPrevStatuses ...task.Status) error {
	query := `
		UPDATE tasks
		SET assigned_master_id = $1, title = $2, description = $3, category = $4,
		    city = $5, district = $6, contact_phone = $7, work_address = $8,
		    budget = $9, status = $10, updated_at = $11
		WHERE id = $12
	`
	args := []any{
		t.AssignedMasterID, t.Title, t.Description, t.Category,
		t.City, t.District, t.ContactPhone, t.WorkAddress,
		t.Budget, t.Status, t.UpdatedAt, t.ID,
	}

	if len(allowedPrevStatuses) > 0 {
		statuses := make([]string, len(allowedPrevStatuses))
		for i, s := range allowedPrevStatuses {
			statuses[i] = string(s)
		}
		query += ` AND status = ANY($13)`
		args = append(args, statuses)
	}

	cmdTag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return mapTaskPgError(err, "update task")
	}
	if cmdTag.RowsAffected() == 0 {
		if len(allowedPrevStatuses) > 0 {
			var exists bool
			if err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM tasks WHERE id = $1)`, t.ID).Scan(&exists); err == nil && exists {
				return task.ErrInvalidStatus
			}
		}
		return task.ErrTaskNotFound
	}
	return nil
}

func (r *TaskRepo) CompleteAndIncrementMasterCounter(ctx context.Context, t *task.Task) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin complete task tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Блокируем строку задачи, чтобы исключить гонку при параллельных /complete
	var currentStatus task.Status
	var assignedMasterID *int64
	if err := tx.QueryRow(
		ctx,
		`SELECT status, assigned_master_id FROM tasks WHERE id = $1 FOR UPDATE`,
		t.ID,
	).Scan(&currentStatus, &assignedMasterID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return task.ErrTaskNotFound
		}
		return fmt.Errorf("lock task for completion: %w", err)
	}

	if assignedMasterID == nil {
		return task.ErrMasterNotAssigned
	}
	if currentStatus != task.StatusInProgress && currentStatus != task.StatusPendingCompletion {
		return task.ErrInvalidStatus
	}

	cmdTag, err := tx.Exec(
		ctx,
		`UPDATE tasks SET status = $1, updated_at = $2 WHERE id = $3 AND status IN ('in_progress', 'pending_completion')`,
		t.Status, t.UpdatedAt, t.ID,
	)
	if err != nil {
		return mapTaskPgError(err, "complete task")
	}
	if cmdTag.RowsAffected() == 0 {
		return task.ErrInvalidStatus
	}

	masterTag, err := tx.Exec(
		ctx,
		`UPDATE master_profiles SET completed_orders_count = completed_orders_count + 1, updated_at = $1 WHERE id = $2`,
		t.UpdatedAt, *assignedMasterID,
	)
	if err != nil {
		return fmt.Errorf("increment master completed_orders_count: %w", err)
	}
	if masterTag.RowsAffected() == 0 {
		return task.ErrInvalidMaster
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit complete task tx: %w", err)
	}
	return nil
}

func (r *TaskRepo) Delete(ctx context.Context, id int64, expectedStatuses ...task.Status) ([]string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin delete task tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var currentStatus task.Status
	var assignedMasterID *int64
	if err := tx.QueryRow(
		ctx,
		`SELECT status, assigned_master_id FROM tasks WHERE id = $1 FOR UPDATE`,
		id,
	).Scan(&currentStatus, &assignedMasterID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, task.ErrTaskNotFound
		}
		return nil, fmt.Errorf("lock task for delete: %w", err)
	}

	if len(expectedStatuses) > 0 {
		allowed := false
		for _, s := range expectedStatuses {
			if currentStatus == s {
				allowed = true
				break
			}
		}
		if !allowed {
			return nil, task.ErrInvalidStatus
		}
	}

	// Собираем storage_key всех фото до каскадного удаления задачи
	photoRows, err := tx.Query(ctx, `SELECT storage_key FROM task_photos WHERE task_id = $1`, id)
	if err != nil {
		return nil, fmt.Errorf("select task photo keys before delete: %w", err)
	}
	var storageKeys []string
	for photoRows.Next() {
		var key string
		if err := photoRows.Scan(&key); err != nil {
			photoRows.Close()
			return nil, fmt.Errorf("scan task photo key: %w", err)
		}
		storageKeys = append(storageKeys, key)
	}
	if err := photoRows.Err(); err != nil {
		photoRows.Close()
		return nil, fmt.Errorf("iterate task photo keys: %w", err)
	}
	photoRows.Close()

	// Строго до DELETE FROM tasks (каскадного удаления master_reviews) блокируем master_profiles FOR UPDATE,
	// чтобы порядок блокировок совпадал с ReviewRepo и исключал deadlock (40P01).
	if currentStatus == task.StatusCompleted && assignedMasterID != nil {
		if err := lockMasterProfileTx(ctx, tx, *assignedMasterID); err != nil {
			return nil, fmt.Errorf("lock assigned master profile before task delete: %w", err)
		}
	}

	cmdTag, err := tx.Exec(ctx, `DELETE FROM tasks WHERE id = $1`, id)
	if err != nil {
		return nil, fmt.Errorf("delete task: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return nil, task.ErrTaskNotFound
	}

	if currentStatus == task.StatusCompleted && assignedMasterID != nil {
		if _, err := tx.Exec(
			ctx,
			`UPDATE master_profiles SET completed_orders_count = GREATEST(completed_orders_count - 1, 0), updated_at = NOW() WHERE id = $1`,
			*assignedMasterID,
		); err != nil {
			return nil, fmt.Errorf("decrement master completed_orders_count: %w", err)
		}
		if err := recalculateMasterRatingTx(ctx, tx, *assignedMasterID); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit delete task tx: %w", err)
	}
	return storageKeys, nil
}

func (r *TaskRepo) AddPhotos(ctx context.Context, taskID, actorID int64, actorRole user.Role, photos []*task.Photo) error {
	if len(photos) == 0 {
		return nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin add task photos tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var t task.Task
	if err := tx.QueryRow(
		ctx,
		`SELECT id, author_id, status FROM tasks WHERE id = $1 FOR UPDATE`,
		taskID,
	).Scan(&t.ID, &t.AuthorID, &t.Status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return task.ErrTaskNotFound
		}
		return fmt.Errorf("lock task for adding photos: %w", err)
	}

	if err := t.CanManagePhotos(actorID, actorRole); err != nil {
		return err
	}

	var existingCount, maxSortOrder int
	if err := tx.QueryRow(
		ctx,
		`SELECT COUNT(*)::int, COALESCE(MAX(sort_order), -1)::int FROM task_photos WHERE task_id = $1`,
		taskID,
	).Scan(&existingCount, &maxSortOrder); err != nil {
		return fmt.Errorf("count existing task photos: %w", err)
	}

	if existingCount+len(photos) > task.MaxPhotosPerTask {
		return task.ErrTooManyPhotos
	}

	insertQuery := `
		INSERT INTO task_photos (task_id, storage_key, mime_type, size_bytes, sort_order, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	for i, p := range photos {
		p.TaskID = taskID
		p.SortOrder = maxSortOrder + 1 + i
		if err := tx.QueryRow(
			ctx, insertQuery,
			p.TaskID, p.StorageKey, p.MimeType, p.SizeBytes, p.SortOrder, p.CreatedAt,
		).Scan(&p.ID); err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23514" {
				return task.ErrUnsupportedPhotoType
			}
			return fmt.Errorf("insert task photo: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit add task photos tx: %w", err)
	}
	return nil
}

func (r *TaskRepo) ListPhotosByTaskID(ctx context.Context, taskID int64) ([]*task.Photo, error) {
	query := `
		SELECT id, task_id, storage_key, mime_type, size_bytes, sort_order, created_at
		FROM task_photos
		WHERE task_id = $1
		ORDER BY sort_order ASC, id ASC
	`
	rows, err := r.pool.Query(ctx, query, taskID)
	if err != nil {
		return nil, fmt.Errorf("list task photos: %w", err)
	}
	defer rows.Close()

	var result []*task.Photo
	for rows.Next() {
		var p task.Photo
		if err := rows.Scan(
			&p.ID, &p.TaskID, &p.StorageKey, &p.MimeType, &p.SizeBytes, &p.SortOrder, &p.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan task photo row: %w", err)
		}
		result = append(result, &p)
	}
	return result, rows.Err()
}

func (r *TaskRepo) ListPhotosByTaskIDs(ctx context.Context, taskIDs []int64) (map[int64][]*task.Photo, error) {
	result := make(map[int64][]*task.Photo)
	if len(taskIDs) == 0 {
		return result, nil
	}

	query := `
		SELECT id, task_id, storage_key, mime_type, size_bytes, sort_order, created_at
		FROM task_photos
		WHERE task_id = ANY($1)
		ORDER BY task_id ASC, sort_order ASC, id ASC
	`
	rows, err := r.pool.Query(ctx, query, taskIDs)
	if err != nil {
		return nil, fmt.Errorf("list task photos by task ids: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var p task.Photo
		if err := rows.Scan(
			&p.ID, &p.TaskID, &p.StorageKey, &p.MimeType, &p.SizeBytes, &p.SortOrder, &p.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan task photo row: %w", err)
		}
		result[p.TaskID] = append(result[p.TaskID], &p)
	}
	return result, rows.Err()
}

func (r *TaskRepo) DeletePhoto(ctx context.Context, taskID, photoID int64, expectedStatuses ...task.Status) (*task.Photo, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin delete task photo tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Блокируем родительскую задачу, чтобы исключить TOCTOU с параллельным изменением статуса
	var currentStatus task.Status
	if err := tx.QueryRow(
		ctx,
		`SELECT status FROM tasks WHERE id = $1 FOR UPDATE`,
		taskID,
	).Scan(&currentStatus); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, task.ErrTaskNotFound
		}
		return nil, fmt.Errorf("lock task for deleting photo: %w", err)
	}

	if len(expectedStatuses) > 0 {
		allowed := false
		for _, s := range expectedStatuses {
			if currentStatus == s {
				allowed = true
				break
			}
		}
		if !allowed {
			return nil, task.ErrInvalidStatus
		}
	}

	var deleted task.Photo
	err = tx.QueryRow(
		ctx,
		`DELETE FROM task_photos
		 WHERE id = $1 AND task_id = $2
		 RETURNING id, task_id, storage_key, mime_type, size_bytes, sort_order, created_at`,
		photoID, taskID,
	).Scan(
		&deleted.ID, &deleted.TaskID, &deleted.StorageKey, &deleted.MimeType,
		&deleted.SizeBytes, &deleted.SortOrder, &deleted.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, task.ErrPhotoNotFound
		}
		return nil, fmt.Errorf("delete task photo: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit delete task photo tx: %w", err)
	}
	return &deleted, nil
}

func mapTaskPgError(err error, action string) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23503":
			return task.ErrInvalidMaster
		case "23514":
			return task.ErrInvalidBudget
		}
	}
	return fmt.Errorf("%s: %w", action, err)
}
