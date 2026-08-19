package model

import "time"

// Типы секретов. Строковые константы — легко расширяются
// (в будущем можно добавить otp, ssh_key и т.п. без миграции).
const (
	SecretTypeLogin  = "login"
	SecretTypeText   = "text"
	SecretTypeBinary = "binary"
	SecretTypeCard   = "card"
)

// Secret — универсальная запись пользовательского секрета.
// Data хранится в зашифрованном виде (E2E): сервер только пересылает blob
// и не может расшифровать его без ключа, известного только клиенту.
type Secret struct {
	ID        string    `json:"id"`
	UserID    int64     `json:"user_id"`
	Type      string    `json:"type"`
	Data      []byte    `json:"data"`
	Meta      string    `json:"meta"`
	Revision  int       `json:"revision"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
