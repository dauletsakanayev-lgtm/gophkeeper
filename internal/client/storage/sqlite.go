// Package storage реализует локальное хранилище GophKeeper на клиенте:
// SQLite-кэш секретов для оффлайн-работы и служебной метаинформации
// (текущий пользователь, метка последней синхронизации).
package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite" // регистрация pure-Go SQLite-драйвера
)

// DefaultDBPath возвращает путь к локальному файлу БД клиента.
// Порядок: env GOPHKEEPER_HOME → ~/.gophkeeper/. Если ни то, ни другое
// не работает — возвращает "gophkeeper.db" рядом с бинарником (CWD).
func DefaultDBPath() string {
	if v := os.Getenv("GOPHKEEPER_HOME"); v != "" {
		return filepath.Join(v, "local.db")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "gophkeeper.db"
	}
	return filepath.Join(home, ".gophkeeper", "local.db")
}

// Open открывает (и при необходимости создаёт) файл SQLite-БД по пути path,
// применяет схему и возвращает готовое *sql.DB. Владелец обязан вызывать Close.
func Open(path string) (*sql.DB, error) {
	// Создаём каталог, если его нет.
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return nil, fmt.Errorf("mkdir %s: %w", dir, err)
		}
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %s: %w", path, err)
	}
	if err := applySchema(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

// applySchema создаёт таблицы (CREATE IF NOT EXISTS), безопасен для повторных вызовов.
func applySchema(db *sql.DB) error {
	const schema = `
		CREATE TABLE IF NOT EXISTS local_secrets (
			id         TEXT PRIMARY KEY,
			type       TEXT    NOT NULL,
			data       BLOB    NOT NULL,
			meta       TEXT    NOT NULL DEFAULT '',
			revision   INTEGER NOT NULL,
			created_at TEXT    NOT NULL,
			updated_at TEXT    NOT NULL,
			-- dirty=1 означает, что локальная копия свежее серверной
			-- и должна быть отправлена при следующем sync push.
			dirty      INTEGER NOT NULL DEFAULT 0
		);
		CREATE INDEX IF NOT EXISTS idx_local_secrets_dirty ON local_secrets(dirty);

		CREATE TABLE IF NOT EXISTS meta (
			key   TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);
	`
	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}
	return nil
}
