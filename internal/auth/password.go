// Package auth реализует регистрацию, аутентификацию и выдачу JWT-токенов.
package auth

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// DefaultCost - рекомендуемое значение bcrypt.Cost на 2024+.
// В тестах разумно понижать до bcrypt.MinCost (4) ради скорости.
const DefaultCost = 12

// ErrPasswordMismatch возвращается при несовпадении пароля и хеша.
// Отдельная типизированная ошибка позволяет отличить ее от инфраструктурных
// сбоев и не отвечать клиенту ничего лишнего.
var ErrPasswordMismatch = errors.New("password mismatch")

// HashPassword хеширует пароль через bcrypt с заданной сложностью.
func HashPassword(password string, cost int) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", fmt.Errorf("bcrypt hash: %w", err)
	}
	return string(h), err
}

// ComparePassword сверять пароль с bcrypt-хешем.
// Возвращает ErrPasswordMismatch при несовпадении и обёрнутую ошибку иначе.
func ComparePassword(hash, password string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return ErrPasswordMismatch
		}
		return fmt.Errorf("bcrypt compare: %w", err)
	}
	return nil
}
