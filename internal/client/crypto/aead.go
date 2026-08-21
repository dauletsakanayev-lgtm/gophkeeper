package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
)

// nonceSize — фиксированный размер nonce для AES-GCM.
// Стандартное значение (12 байт) — единственно поддерживаемое GCM в crypto/cipher.
const nonceSize = 12

// ErrInvalidKey — ключ не 32 байта (не AES-256).
// ErrCiphertextTooShort — переданные данные короче nonce, расшифровать нечего.
// ErrDecrypt — при расшифровке не совпал MAC (данные битые или ключ не тот).
var (
	ErrInvalidKey         = errors.New("key must be 32 bytes for AES-256")
	ErrCiphertextTooShort = errors.New("ciphertext too short")
	ErrDecrypt            = errors.New("decrypt failed")
)

// Encrypt шифрует plaintext AES-256-GCM с указанным 32-байтным ключом.
// Возвращает: nonce(12) || ciphertext || tag(16) — всё одним слайсом,
// готовым к передаче на сервер / записи в БД.
//
// Каждый вызов генерирует свежий random nonce (12 байт) — повтор nonce
// с тем же ключом ломает confidentiality GCM.
func Encrypt(key, plaintext []byte) ([]byte, error) {
	if len(key) != keyLen {
		return nil, ErrInvalidKey
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}
	nonce := make([]byte, nonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("nonce: %w", err)
	}
	// gcm.Seal(dst, nonce, plaintext, additionalData)
	// Передаём nonce как первые байты dst — Seal допишет ciphertext+tag.
	sealed := gcm.Seal(nonce, nonce, plaintext, nil)
	return sealed, nil
}

// Decrypt расшифровывает данные, произведённые Encrypt.
// Возвращает ErrDecrypt при несовпадении MAC — это либо битый ciphertext,
// либо неверный ключ (неверный master password). Различить снаружи их нельзя.
func Decrypt(key, sealed []byte) ([]byte, error) {
	if len(key) != keyLen {
		return nil, ErrInvalidKey
	}
	if len(sealed) < nonceSize {
		return nil, ErrCiphertextTooShort
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}
	nonce, ciphertext := sealed[:nonceSize], sealed[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrDecrypt
	}
	return plaintext, nil
}
