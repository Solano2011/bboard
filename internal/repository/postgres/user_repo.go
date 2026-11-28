package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"bboard/internal/domain/user"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ user.Repository = (*UserRepo)(nil)

type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

func (r *UserRepo) Create(ctx context.Context, u *user.User) error {
	if u.Role == "" {
		u.Role = user.RoleUser
	}

	query := `
		INSERT INTO users (name, email, phone, role, password_hash, email_verified_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`
	err := r.pool.QueryRow(
		ctx,
		query,
		u.Name,
		u.Email,
		u.Phone,
		u.Role,
		u.PasswordHash,
		u.EmailVerifiedAt,
		u.CreatedAt,
		u.UpdatedAt,
	).Scan(&u.ID)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return user.ErrUserAlreadyExist
		}
		return fmt.Errorf("insert user into postgres: %w", err)
	}

	return nil
}

func (r *UserRepo) GetByID(ctx context.Context, id int64) (*user.User, error) {
	query := `
		SELECT id, name, email, phone, role, password_hash, email_verified_at, created_at, updated_at
		FROM users
		WHERE id = $1
	`
	return r.scanUser(r.pool.QueryRow(ctx, query, id))
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	query := `
		SELECT id, name, email, phone, role, password_hash, email_verified_at, created_at, updated_at
		FROM users
		WHERE email = $1
	`
	return r.scanUser(r.pool.QueryRow(ctx, query, normalizedEmail))
}

func (r *UserRepo) scanUser(row pgx.Row) (*user.User, error) {
	var u user.User
	err := row.Scan(
		&u.ID,
		&u.Name,
		&u.Email,
		&u.Phone,
		&u.Role,
		&u.PasswordHash,
		&u.EmailVerifiedAt,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, user.ErrUserNotFound
		}
		return nil, fmt.Errorf("select user from postgres: %w", err)
	}
	return &u, nil
}

func (r *UserRepo) List(ctx context.Context, search string, limit, offset int) ([]*user.User, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	var args []any
	query := `
		SELECT id, name, email, phone, role, password_hash, email_verified_at, created_at, updated_at
		FROM users
	`
	search = strings.TrimSpace(search)
	if search != "" {
		escaped := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(search)
		query += ` WHERE name ILIKE $1 ESCAPE '\' OR email ILIKE $1 ESCAPE '\'`
		args = append(args, "%"+escaped+"%")
		query += ` ORDER BY id ASC LIMIT $2 OFFSET $3`
		args = append(args, limit, offset)
	} else {
		query += ` ORDER BY id ASC LIMIT $1 OFFSET $2`
		args = append(args, limit, offset)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list users from postgres: %w", err)
	}
	defer rows.Close()

	var result []*user.User
	for rows.Next() {
		var u user.User
		if err := rows.Scan(
			&u.ID,
			&u.Name,
			&u.Email,
			&u.Phone,
			&u.Role,
			&u.PasswordHash,
			&u.EmailVerifiedAt,
			&u.CreatedAt,
			&u.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan user row: %w", err)
		}
		result = append(result, &u)
	}
	return result, rows.Err()
}

func (r *UserRepo) Update(ctx context.Context, u *user.User) error {
	query := `
		UPDATE users
		SET name = $1,
		    phone = $2,
		    password_hash = $3,
		    updated_at = $4
		WHERE id = $5
	`
	cmdTag, err := r.pool.Exec(ctx, query, u.Name, u.Phone, u.PasswordHash, u.UpdatedAt, u.ID)
	if err != nil {
		return fmt.Errorf("update user in postgres: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return user.ErrUserNotFound
	}
	return nil
}

func (r *UserRepo) UpdateRole(ctx context.Context, id int64, role user.Role, updatedAt time.Time) error {
	query := `
		UPDATE users
		SET role = $1,
		    updated_at = $2
		WHERE id = $3 AND role <> 'admin'
	`
	cmdTag, err := r.pool.Exec(ctx, query, role, updatedAt, id)
	if err != nil {
		return fmt.Errorf("update user role in postgres: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		var currentRole user.Role
		checkErr := r.pool.QueryRow(ctx, `SELECT role FROM users WHERE id = $1`, id).Scan(&currentRole)
		if errors.Is(checkErr, pgx.ErrNoRows) {
			return user.ErrUserNotFound
		}
		if checkErr != nil {
			return fmt.Errorf("check user role after update: %w", checkErr)
		}
		if currentRole == user.RoleAdmin {
			return user.ErrForbiddenRoleChange
		}
		return user.ErrUserNotFound
	}
	return nil
}

func (r *UserRepo) PromoteToAdminByEmail(ctx context.Context, email string) error {
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	if normalizedEmail == "" {
		return nil
	}

	_, err := r.pool.Exec(
		ctx,
		`UPDATE users SET role = 'admin', updated_at = NOW() WHERE email = $1 AND role <> 'admin'`,
		normalizedEmail,
	)
	if err != nil {
		return fmt.Errorf("promote user to admin by email: %w", err)
	}
	return nil
}
