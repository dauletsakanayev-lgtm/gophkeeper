package session

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/term"
)

// ErrPasswordMismatch — введённый password ≠ подтверждению (при регистрации).
var ErrPasswordMismatch = errors.New("passwords do not match")

// ReadPassword скрыто читает пароль из stdin (без эха в консоль).
// Prompt печатается в stderr, чтобы не смешиваться с pipe'ом на stdout.
func ReadPassword(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	pw, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr) // перевод строки после скрытого ввода
	if err != nil {
		return "", fmt.Errorf("read password: %w", err)
	}
	return string(pw), nil
}

// ReadPasswordConfirm читает пароль дважды и проверяет совпадение.
// Используется при первичной регистрации (login/master password), чтобы
// не запомнить неверный пароль из-за опечатки.
func ReadPasswordConfirm(prompt string) (string, error) {
	pw, err := ReadPassword(prompt)
	if err != nil {
		return "", err
	}
	confirm, err := ReadPassword("Confirm: ")
	if err != nil {
		return "", err
	}
	if pw != confirm {
		return "", ErrPasswordMismatch
	}
	return pw, nil
}
