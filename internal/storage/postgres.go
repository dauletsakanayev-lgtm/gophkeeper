// Package storage реализует слой хранения GophKeeper: подключение к PostgreSQL,
// применение миграций и репозитории для доменных сущности.

package storage

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"

	"github.com/dauletsakanayev-lgtm/gophkeeper/internal/storage/migrations"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// Open открывает соединение с PostgreSQL по указанному DNS и проверяет
// доступность БД короткой пингой. Возвращает готовый *sql.DB, которым владелец
// обязан управлять (Close по завершении работ).
func Open(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return db, nil
}

// Migrate применяет все ещё не применённые миграции из встроенной FS.
// Использует собственную мини-реализацию (без внешних зависимостей):
//   - таблица schema_migrations хранит применённые версии;
//   - миграция = один .up.sql файл, версия = имя без .up.sql;
//   - файлы применяются в алфавитном порядке (см. префикс 0001_, 0002_ ...).
//
// Идемпотентно: повторный вызов на актуальной схеме ничего не меняет.
func Migrate(db *sql.DB) error {
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations(
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	entries, err := fs.ReadDir(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	var files []string
	for _, e := range entries {
		name := e.Name()
		if !e.IsDir() && strings.HasSuffix(name, ".up.sql") {
			files = append(files, name)
		}
	}
	sort.Strings(files)

	for _, name := range files {
		version := strings.TrimSuffix(name, "up.sql")

		var count bool
		if err := db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM schema_migrations WHERE version = $1`,
			version).Scan(&count); err != nil {
			return fmt.Errorf("check migration %s: %w", version, err)
		}
		if count {
			continue
		}
		context, err := migrations.FS.ReadFile(name)
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}

		if _, err := db.ExecContext(ctx, string(context)); err != nil {
			return fmt.Errorf("apply %s: %w", name, err)
		}

		if _, err := db.ExecContext(ctx,
			`INSERT INTO schema_migrations (version) VALUES ($1)`, version); err != nil {
			return fmt.Errorf("record migration %s: %w", version, err)
		}
	}
	return nil
}
