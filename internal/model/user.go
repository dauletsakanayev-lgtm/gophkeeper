// Package model содержит доменные типы GopherKeeper, общие для сервера,
// хранилище и HTTP-транспорта.
package model

import "time"

// User - учетная запись пользователя.
// PasswordHash хранит bcrypt-хеш; сам пороль в системе не хранится.
type User struct {
	ID           int64     `json:"id"`
	Login        string    `json:"login"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}
