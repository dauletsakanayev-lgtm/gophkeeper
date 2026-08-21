// Package crypto реализует клиентскую сторону end-to-end шифрования
// GophKeeper: вывод ключа из master password и AEAD-шифрование
// произвольных данных. Работает только в памяти клиента —
// master password никогда не сохраняется и не покидает устройство.
package crypto

import (
	"crypto/sha256"

	"golang.org/x/crypto/argon2"
)

// Параметры Argon2id — рекомендация OWASP на 2024+ для password managers.
// time=1 memory=64MB threads=4 keyLen=32 → ~50–100ms на средней машине.
// Слишком дорого = раздражает при каждом login, слишком дёшево = легко брутить.
const (
	argonTime    = 1
	argonMemory  = 64 * 1024 // 64 MB
	argonThreads = 4
	keyLen       = 32 // 32 байта = 256 бит → AES-256
)

// DeriveKey выводит 32-байтный симметричный ключ из master password и login.
//
// Salt = SHA-256(login)[:16] — детерминированно, уникально per user.
// Такой salt позволяет любому клиенту того же пользователя вывести тот же
// ключ, зная только master password + login (без обращения к серверу за salt).
// В обмен теряется stateless-случайность salt, но per-user рейнбоу-таблиц
// это не спасает: атакующему нужны отдельные таблицы под каждый login.
func DeriveKey(masterPassword, login string) []byte {
	salt := userSalt(login)
	return argon2.IDKey(
		[]byte(masterPassword),
		salt,
		argonTime,
		argonMemory,
		argonThreads,
		keyLen,
	)
}

// userSalt — детерминированный per-user salt из первых 16 байт SHA-256(login).
// Login публично известен (лежит в JWT), поэтому secrecy salt не даёт.
// Функция — только чтобы каждый пользователь имел собственный salt.
func userSalt(login string) []byte {
	h := sha256.Sum256([]byte(login))
	return h[:16]
}
