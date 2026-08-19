package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/dauletsakanayev-lgtm/gophkeeper/internal/model"
)

// ErrSecretNotFound — секрет по указанному id отсутствует у этого пользователя.
// ErrRevisionMismatch — клиент пытается обновить устаревшую версию секрета.
var (
	ErrSecretNotFound   = errors.New("secret not found")
	ErrRevisionMismatch = errors.New("revision mismatch")
)

// SecretRepository — контракт хранилища секретов.
// Все методы принимают userID и работают только с секретами этого пользователя —
// авторизация на уровне запроса.
type SecretRepository interface {
	Create(ctx context.Context, userID int64, secretType string, data []byte, meta string) (*model.Secret, error)
	Get(ctx context.Context, userID int64, id string) (*model.Secret, error)
	List(ctx context.Context, userID int64) ([]*model.Secret, error)
	Update(ctx context.Context, userID int64, id string, expectedRevision int, data []byte, meta string) (*model.Secret, error)
	Delete(ctx context.Context, userID int64, id string) error
}

// PostgresSecretRepo — реализация SecretRepository поверх PostgreSQL.
type PostgresSecretRepo struct {
	db *sql.DB
}

// NewPostgresSecretRepo конструирует репозиторий, принимая уже открытое соединение.
func NewPostgresSecretRepo(db *sql.DB) *PostgresSecretRepo {
	return &PostgresSecretRepo{db: db}
}

// Create сохраняет новый секрет и возвращает его со всеми серверными полями
// (id, revision=1, created_at, updated_at).
func (r *PostgresSecretRepo) Create(ctx context.Context, userID int64, secretType string, data []byte, meta string) (*model.Secret, error) {
	const q = `
		INSERT INTO secrets (user_id, type, data, meta)
		VALUES ($1, $2, $3, $4)
		RETURNING id, user_id, type, data, meta, revision, created_at, updated_at
	`
	s := &model.Secret{}
	err := r.db.QueryRowContext(ctx, q, userID, secretType, data, meta).
		Scan(&s.ID, &s.UserID, &s.Type, &s.Data, &s.Meta, &s.Revision, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert secret: %w", err)
	}
	return s, nil
}

// Get возвращает секрет по id, если он принадлежит указанному пользователю.
// Возвращает ErrSecretNotFound в остальных случаях (не найден или чужой).
func (r *PostgresSecretRepo) Get(ctx context.Context, userID int64, id string) (*model.Secret, error) {
	const q = `
		SELECT id, user_id, type, data, meta, revision, created_at, updated_at
		FROM secrets
		WHERE id = $1 AND user_id = $2
	`
	s := &model.Secret{}
	err := r.db.QueryRowContext(ctx, q, id, userID).
		Scan(&s.ID, &s.UserID, &s.Type, &s.Data, &s.Meta, &s.Revision, &s.CreatedAt, &s.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrSecretNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("select secret: %w", err)
	}
	return s, nil
}

// List возвращает все секреты пользователя, отсортированные по updated_at DESC.
func (r *PostgresSecretRepo) List(ctx context.Context, userID int64) ([]*model.Secret, error) {
	const q = `
		SELECT id, user_id, type, data, meta, revision, created_at, updated_at
		FROM secrets
		WHERE user_id = $1
		ORDER BY updated_at DESC
	`
	rows, err := r.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("query secrets: %w", err)
	}
	defer rows.Close()

	var result []*model.Secret
	for rows.Next() {
		s := &model.Secret{}
		if err := rows.Scan(&s.ID, &s.UserID, &s.Type, &s.Data, &s.Meta, &s.Revision, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan secret: %w", err)
		}
		result = append(result, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}
	return result, nil
}

// Update обновляет data/meta секрета только если expectedRevision совпадает
// с текущим на сервере (optimistic locking). Возвращает:
//   - ErrSecretNotFound, если id/user_id не найдены;
//   - ErrRevisionMismatch, если revision клиента устарел (кто-то обновил раньше);
//   - обновлённый секрет с revision+1 и свежим updated_at при успехе.
func (r *PostgresSecretRepo) Update(ctx context.Context, userID int64, id string, expectedRevision int, data []byte, meta string) (*model.Secret, error) {
	const q = `
		UPDATE secrets
		SET data = $1, meta = $2, revision = revision + 1, updated_at = NOW()
		WHERE id = $3 AND user_id = $4 AND revision = $5
		RETURNING id, user_id, type, data, meta, revision, created_at, updated_at
	`
	s := &model.Secret{}
	err := r.db.QueryRowContext(ctx, q, data, meta, id, userID, expectedRevision).
		Scan(&s.ID, &s.UserID, &s.Type, &s.Data, &s.Meta, &s.Revision, &s.CreatedAt, &s.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		// UPDATE не задел строк — либо секрет не наш/не найден, либо revision устарел.
		// Проверяем что именно случилось, чтобы вернуть точную ошибку.
		return nil, r.classifyUpdateMiss(ctx, userID, id)
	}
	if err != nil {
		return nil, fmt.Errorf("update secret: %w", err)
	}
	return s, nil
}

// classifyUpdateMiss различает "секрет не найден" от "revision устарел" —
// один SELECT после несостоявшегося UPDATE.
func (r *PostgresSecretRepo) classifyUpdateMiss(ctx context.Context, userID int64, id string) error {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM secrets WHERE id = $1 AND user_id = $2`, id, userID).
		Scan(&count)
	if err != nil {
		return fmt.Errorf("classify update miss: %w", err)
	}
	if count == 0 {
		return ErrSecretNotFound
	}
	return ErrRevisionMismatch
}

// Delete удаляет секрет пользователя. Возвращает ErrSecretNotFound, если
// строки не тронуты (id/user_id не найдены).
func (r *PostgresSecretRepo) Delete(ctx context.Context, userID int64, id string) error {
	const q = `DELETE FROM secrets WHERE id = $1 AND user_id = $2`
	res, err := r.db.ExecContext(ctx, q, id, userID)
	if err != nil {
		return fmt.Errorf("delete secret: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return ErrSecretNotFound
	}
	return nil
}
