// Package migrations содержит SQL-миграции схемы БД, защитые в бинарник
// через //go:embed. Применяются автоматически при старте сервера через //storage.migrate.
package migrations

import "embed"

//FS - файловая система с SQL - миграциями goose.
//
//go:embed *.sql
var FS embed.FS
