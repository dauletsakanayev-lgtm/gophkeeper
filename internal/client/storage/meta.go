package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// Ключи в таблице meta. Хранятся как строки: клиент интерпретирует по месту.
const (
	MetaUserID     = "user_id"
	MetaLogin      = "login"
	MetaLastSyncAt = "last_sync_at"
)

// GetMeta возвращает значение по ключу или пустую строку + false, если ключа нет.
func GetMeta(ctx context.Context, db *sql.DB, key string) (string, bool, error) {
	var v string
	err := db.QueryRowContext(ctx, `SELECT value FROM meta WHERE key = ?`, key).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("get meta %s: %w", key, err)
	}
	return v, true, nil
}

// SetMeta вставляет или обновляет значение по ключу (idempotent).
func SetMeta(ctx context.Context, db *sql.DB, key, value string) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO meta (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`, key, value)
	if err != nil {
		return fmt.Errorf("set meta %s: %w", key, err)
	}
	return nil
}
