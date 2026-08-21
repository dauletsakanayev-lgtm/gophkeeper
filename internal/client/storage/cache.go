package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// LocalSecret — запись локального кэша. Data всегда шифрованная (E2E),
// сервер и локальная БД видят только blob.
type LocalSecret struct {
	ID        string
	Type      string
	Data      []byte
	Meta      string
	Revision  int
	CreatedAt time.Time
	UpdatedAt time.Time
	Dirty     bool
}

// ErrLocalNotFound — секрет с указанным id отсутствует в локальном кэше.
var ErrLocalNotFound = errors.New("local secret not found")

// Cache — локальный кэш секретов поверх SQLite.
type Cache struct {
	db *sql.DB
}

// NewCache возвращает Cache, использующий переданное соединение (уже открытое).
func NewCache(db *sql.DB) *Cache {
	return &Cache{db: db}
}

// Upsert вставляет или обновляет запись по id (принят с сервера — dirty=0,
// либо создан локально — dirty=1). Полное содержимое заменяется.
func (c *Cache) Upsert(ctx context.Context, s LocalSecret) error {
	const q = `
		INSERT INTO local_secrets (id, type, data, meta, revision, created_at, updated_at, dirty)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			type       = excluded.type,
			data       = excluded.data,
			meta       = excluded.meta,
			revision   = excluded.revision,
			updated_at = excluded.updated_at,
			dirty      = excluded.dirty
	`
	dirty := 0
	if s.Dirty {
		dirty = 1
	}
	_, err := c.db.ExecContext(ctx, q,
		s.ID, s.Type, s.Data, s.Meta, s.Revision,
		s.CreatedAt.Format(time.RFC3339Nano),
		s.UpdatedAt.Format(time.RFC3339Nano),
		dirty,
	)
	if err != nil {
		return fmt.Errorf("upsert local secret: %w", err)
	}
	return nil
}

// Get возвращает секрет по id или ErrLocalNotFound.
func (c *Cache) Get(ctx context.Context, id string) (*LocalSecret, error) {
	const q = `
		SELECT id, type, data, meta, revision, created_at, updated_at, dirty
		FROM local_secrets WHERE id = ?
	`
	s := &LocalSecret{}
	var createdStr, updatedStr string
	var dirty int
	err := c.db.QueryRowContext(ctx, q, id).
		Scan(&s.ID, &s.Type, &s.Data, &s.Meta, &s.Revision, &createdStr, &updatedStr, &dirty)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrLocalNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("select local secret: %w", err)
	}
	s.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdStr)
	s.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedStr)
	s.Dirty = dirty != 0
	return s, nil
}

// List возвращает все локальные секреты, отсортированные по updated_at DESC.
func (c *Cache) List(ctx context.Context) ([]*LocalSecret, error) {
	const q = `
		SELECT id, type, data, meta, revision, created_at, updated_at, dirty
		FROM local_secrets ORDER BY updated_at DESC
	`
	rows, err := c.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("query local secrets: %w", err)
	}
	defer rows.Close()

	var result []*LocalSecret
	for rows.Next() {
		s := &LocalSecret{}
		var createdStr, updatedStr string
		var dirty int
		if err := rows.Scan(&s.ID, &s.Type, &s.Data, &s.Meta, &s.Revision, &createdStr, &updatedStr, &dirty); err != nil {
			return nil, fmt.Errorf("scan local secret: %w", err)
		}
		s.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdStr)
		s.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedStr)
		s.Dirty = dirty != 0
		result = append(result, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}
	return result, nil
}

// ListDirty возвращает только записи, помеченные dirty=1 —
// это очередь на отправку при sync push.
func (c *Cache) ListDirty(ctx context.Context) ([]*LocalSecret, error) {
	const q = `
		SELECT id, type, data, meta, revision, created_at, updated_at, dirty
		FROM local_secrets WHERE dirty = 1 ORDER BY updated_at ASC
	`
	rows, err := c.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("query dirty local secrets: %w", err)
	}
	defer rows.Close()

	var result []*LocalSecret
	for rows.Next() {
		s := &LocalSecret{}
		var createdStr, updatedStr string
		var dirty int
		if err := rows.Scan(&s.ID, &s.Type, &s.Data, &s.Meta, &s.Revision, &createdStr, &updatedStr, &dirty); err != nil {
			return nil, fmt.Errorf("scan dirty: %w", err)
		}
		s.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdStr)
		s.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedStr)
		s.Dirty = dirty != 0
		result = append(result, s)
	}
	return result, rows.Err()
}

// ClearDirty снимает флаг dirty после успешного push на сервер.
func (c *Cache) ClearDirty(ctx context.Context, id string) error {
	_, err := c.db.ExecContext(ctx,
		`UPDATE local_secrets SET dirty = 0 WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("clear dirty: %w", err)
	}
	return nil
}

// Delete удаляет запись из локального кэша (например, после DELETE на сервере
// в ответ на pull, обнаруживший удаление).
func (c *Cache) Delete(ctx context.Context, id string) error {
	_, err := c.db.ExecContext(ctx,
		`DELETE FROM local_secrets WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete local secret: %w", err)
	}
	return nil
}
