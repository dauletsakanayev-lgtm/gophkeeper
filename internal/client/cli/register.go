package cli

import (
	"fmt"

	"github.com/dauletsakanayev-lgtm/gophkeeper/internal/client/api"
	"github.com/dauletsakanayev-lgtm/gophkeeper/internal/client/session"
	"github.com/spf13/cobra"
)

var registerCmd = &cobra.Command{
	Use:   "register <login>",
	Short: "Зарегистрировать нового пользователя",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		login := args[0]
		srv := serverOrDefault()

		password, err := session.ReadPasswordConfirm("Password: ")
		if err != nil {
			return fmt.Errorf("password input: %w", err)
		}

		// Master password запрашиваем сразу, чтобы пользователь точно его
		// запомнил (двойной ввод с проверкой). Сам master password серверу
		// не отправляется — используется локально для шифрования секретов.
		if _, err := session.ReadPasswordConfirm("Master password (used for E2E encryption): "); err != nil {
			return fmt.Errorf("master password input: %w", err)
		}

		client := api.New(srv)
		token, err := client.Register(login, password)
		if err != nil {
			return fmt.Errorf("register: %w", err)
		}

		if err := session.Save(session.DefaultPath(), &session.Session{
			ServerURL: srv,
			Login:     login,
			Token:     token,
		}); err != nil {
			return fmt.Errorf("save session: %w", err)
		}

		fmt.Printf("registered as %q, session stored at %s\n", login, session.DefaultPath())
		return nil
	},
}

// serverOrDefault возвращает адрес сервера в порядке приоритета:
// 1) флаг --server, 2) URL из существующей session, 3) defaultServer.
func serverOrDefault() string {
	if serverURL != "" {
		return serverURL
	}
	s, err := session.Load(session.DefaultPath())
	if err == nil && s.ServerURL != "" {
		return s.ServerURL
	}
	return defaultServer
}
