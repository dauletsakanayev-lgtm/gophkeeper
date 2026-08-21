package cli

import (
	"errors"
	"fmt"

	"github.com/dauletsakanayev-lgtm/gophkeeper/internal/client/api"
	"github.com/dauletsakanayev-lgtm/gophkeeper/internal/client/crypto"
	"github.com/dauletsakanayev-lgtm/gophkeeper/internal/client/session"
)

// mustSession загружает session или возвращает подсказку логиниться.
func mustSession() (*session.Session, error) {
	s, err := session.Load(session.DefaultPath())
	if err != nil {
		if errors.Is(err, session.ErrNoSession) {
			return nil, err
		}
		return nil, fmt.Errorf("load session: %w", err)
	}
	return s, nil
}

// authClient — HTTP-клиент с JWT-токеном из активной session.
func authClient(s *session.Session) *api.Client {
	return api.New(s.ServerURL).WithToken(s.Token)
}

// deriveMasterKey интерактивно спрашивает master password и выводит ключ.
// Ключ живёт только в стеке вызывающей функции — не сохраняется никуда.
func deriveMasterKey(login string) ([]byte, error) {
	pw, err := session.ReadPassword("Master password: ")
	if err != nil {
		return nil, err
	}
	if pw == "" {
		return nil, fmt.Errorf("master password is empty")
	}
	return crypto.DeriveKey(pw, login), nil
}
