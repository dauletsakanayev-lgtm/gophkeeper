// Package session хранит долгоживущее состояние клиента: JWT-токен и адрес
// сервера. Мастер-пароль в session НЕ хранится — вводится при каждой crypto-операции.
package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Session — что клиент помнит между запусками CLI.
type Session struct {
	ServerURL string `json:"server_url"`
	Login     string `json:"login"`
	Token     string `json:"token"`
}

// ErrNoSession — файл сессии отсутствует (пользователь не логинился).
var ErrNoSession = errors.New("no active session; please run: gophkeeper login")

// DefaultPath возвращает путь ~/.gophkeeper/session.json (или GOPHKEEPER_HOME/session.json).
func DefaultPath() string {
	if v := os.Getenv("GOPHKEEPER_HOME"); v != "" {
		return filepath.Join(v, "session.json")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "session.json"
	}
	return filepath.Join(home, ".gophkeeper", "session.json")
}

// Load читает файл сессии. Возвращает ErrNoSession, если файла нет.
func Load(path string) (*Session, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNoSession
		}
		return nil, fmt.Errorf("read session: %w", err)
	}
	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("decode session: %w", err)
	}
	return &s, nil
}

// Save записывает сессию с правами 0600 (только владелец), создаёт каталог при необходимости.
func Save(path string, s *Session) error {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("mkdir %s: %w", dir, err)
		}
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("encode session: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write session: %w", err)
	}
	return nil
}

// Delete удаляет файл сессии (для logout). Отсутствие файла — не ошибка.
func Delete(path string) error {
	err := os.Remove(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}
