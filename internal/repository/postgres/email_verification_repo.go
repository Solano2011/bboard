package postgres

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"strings"
	"time"

	"bboard/internal/domain/auth"
	"bboard/internal/domain/user"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ auth.EmailVerificationRepository = (*EmailVerificationRepo)(nil)

type EmailVerificationRepo struct {
	pool *pgxpool.Pool
}

func NewEmailVerificationRepo(pool *pgxpool.Pool) *EmailVerificationRepo {
	return &EmailVerificationRepo{pool: pool}
}

// SaveAndSendCode атомарно проверяет окно повторной отправки, сохраняет новый код и отправляет письмо до коммита.
func (r *EmailVerificationRepo) SaveAndSendCode(
	ctx context.Context,
	email, codeHash string,
	now, expiresAt time.Time,
	maxAttempts int,
	sendFn func() error,
) error {
	email = strings.ToLower(strings.TrimSpace(email))
	if maxAttempts <= 0 {
		maxAttempts = auth.MaxVerificationAttempts
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin email verification tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// Блокируем email на уровне транзакции, чтобы параллельная первичная отправка не обошла cooldown.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, email); err != nil {
		return fmt.Errorf("acquire advisory lock for email verification: %w", err)
	}

	var lastSentAt time.Time
	err = tx.QueryRow(
		ctx,
		`SELECT last_sent_at FROM email_verification_codes WHERE email = $1 FOR UPDATE`,
		email,
	).Scan(&lastSentAt)
	if err == nil {
		codeState := auth.EmailVerificationCode{LastSentAt: lastSentAt}
		if ok, retrySecs := codeState.CanResend(now); !ok {
			return &auth.ResendCooldownError{RetryAfterSeconds: retrySecs}
		}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("check existing verification code: %w", err)
	}

	upsertQuery := `
		INSERT INTO email_verification_codes (
			email, code_hash, expires_at, last_sent_at, attempts, max_attempts, used_at, created_at
		)
		VALUES ($1, $2, $3, $4, 0, $5, NULL, $6)
		ON CONFLICT (email) DO UPDATE SET
			code_hash = EXCLUDED.code_hash,
			expires_at = EXCLUDED.expires_at,
			last_sent_at = EXCLUDED.last_sent_at,
			attempts = 0,
			max_attempts = EXCLUDED.max_attempts,
			used_at = NULL
	`
	if _, err := tx.Exec(ctx, upsertQuery, email, codeHash, expiresAt, now, maxAttempts, now); err != nil {
		return fmt.Errorf("upsert verification code: %w", err)
	}

	if err := sendFn(); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit email verification tx: %w", err)
	}
	return nil
}

// VerifyCodeAndCreateUser проверяет одноразовый код и создаёт подтверждённого пользователя в одной транзакции.
func (r *EmailVerificationRepo) VerifyCodeAndCreateUser(
	ctx context.Context,
	email, codeHash string,
	now time.Time,
	u *user.User,
) error {
	email = strings.ToLower(strings.TrimSpace(email))
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin verify code and create user tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var rec auth.EmailVerificationCode
	selectQuery := `
		SELECT id, code_hash, expires_at, attempts, max_attempts, used_at
		FROM email_verification_codes
		WHERE email = $1
		FOR UPDATE
	`
	err = tx.QueryRow(ctx, selectQuery, email).Scan(
		&rec.ID,
		&rec.CodeHash,
		&rec.ExpiresAt,
		&rec.Attempts,
		&rec.MaxAttempts,
		&rec.UsedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return auth.ErrVerificationCodeNotFound
		}
		return fmt.Errorf("select verification code for update: %w", err)
	}

	if rec.IsUsed() {
		return auth.ErrVerificationCodeNotFound
	}
	if rec.IsExpired(now) {
		return auth.ErrVerificationCodeExpired
	}
	if rec.IsExhausted() {
		return auth.ErrVerificationCodeExhausted
	}

	if subtle.ConstantTimeCompare([]byte(rec.CodeHash), []byte(codeHash)) != 1 {
		if _, err := tx.Exec(ctx, `UPDATE email_verification_codes SET attempts = attempts + 1 WHERE id = $1`, rec.ID); err != nil {
			return fmt.Errorf("increment verification attempts: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit verification attempt increment: %w", err)
		}
		rec.Attempts++
		if rec.IsExhausted() {
			return auth.ErrVerificationCodeExhausted
		}
		return auth.ErrInvalidVerificationCode
	}

	if _, err := tx.Exec(ctx, `UPDATE email_verification_codes SET used_at = $1 WHERE id = $2`, now, rec.ID); err != nil {
		return fmt.Errorf("mark verification code used: %w", err)
	}

	if u.Role == "" {
		u.Role = user.RoleUser
	}
	if u.EmailVerifiedAt == nil {
		verifiedAt := now
		u.EmailVerifiedAt = &verifiedAt
	}

	insertUserQuery := `
		INSERT INTO users (name, email, phone, role, password_hash, email_verified_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`
	err = tx.QueryRow(
		ctx,
		insertUserQuery,
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
		return fmt.Errorf("insert verified user into postgres: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit verified user tx: %w", err)
	}
	return nil
}
