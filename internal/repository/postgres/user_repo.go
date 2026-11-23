package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"bboard/internal/domain/user"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Проверка на этапе компиляции, что UserRepo полностью реализует интерфейс user.Repository.
var _ user.Repository = (*UserRepo)(nil)

// UserRepo реализует хранение пользователей в PostgreSQL.
type UserRepo struct {
	pool *pgxpool.Pool
}

// NewUserRepo создаёт новый экземпляр репозитория пользователей.
func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{
		pool: pool,
	}
}

// Create вставляет нового пользователя в таблицу users и записывает сгенерированный ID в структуру u.
func (r *UserRepo) Create(ctx context.Context, u *user.User) error {
	query := `
		INSERT INTO users (name, email, phone, password_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	err := r.pool.QueryRow(
		ctx,
		query,
		u.Name,
		u.Email,
		u.Phone,
		u.PasswordHash,
		u.CreatedAt,
		u.UpdatedAt,
	).Scan(&u.ID)

	if err != nil {
		var pgErr *pgconn.PgError
		// 23505 — код ошибки PostgreSQL для нарушения ограничения UNIQUE (unique_violation)
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return user.ErrUserAlreadyExist
		}
		return fmt.Errorf("insert user into postgres: %w", err)
	}

	return nil
}

// GetByID находит пользователя по первичному ключу.
func (r *UserRepo) GetByID(ctx context.Context, id int64) (*user.User, error) {
	query := `
		SELECT id, name, email, phone, password_hash, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var u user.User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&u.ID,
		&u.Name,
		&u.Email,
		&u.Phone,
		&u.PasswordHash,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, user.ErrUserNotFound
		}
		return nil, fmt.Errorf("select user by id from postgres: %w", err)
	}

	return &u, nil
}

// GetByEmail находит пользователя по email.
func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))

	query := `
		SELECT id, name, email, phone, password_hash, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	var u user.User
	err := r.pool.QueryRow(ctx, query, normalizedEmail).Scan(
		&u.ID,
		&u.Name,
		&u.Email,
		&u.Phone,
		&u.PasswordHash,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, user.ErrUserNotFound
		}
		return nil, fmt.Errorf("select user by email from postgres: %w", err)
	}

	return &u, nil
}

// Update обновляет изменяемые поля существующего пользователя.
func (r *UserRepo) Update(ctx context.Context, u *user.User) error {
	query := `
		UPDATE users
		SET name = $1,
		    phone = $2,
		    password_hash = $3,
		    updated_at = $4
		WHERE id = $5
	`

	cmdTag, err := r.pool.Exec(
		ctx,
		query,
		u.Name,
		u.Phone,
		u.PasswordHash,
		u.UpdatedAt,
		u.ID,
	)
	if err != nil {
		return fmt.Errorf("update user in postgres: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return user.ErrUserNotFound
	}

	return nil
}
