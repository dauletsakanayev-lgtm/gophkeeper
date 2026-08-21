package cli

import (
	"fmt"

	"github.com/dauletsakanayev-lgtm/gophkeeper/internal/client/api"
	"github.com/dauletsakanayev-lgtm/gophkeeper/internal/client/session"
	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login <login>",
	Short: "Аутентификация: сохраняет JWT-токен для последующих команд",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		login := args[0]
		srv := serverOrDefault()

		password, err := session.ReadPassword("Password: ")
		if err != nil {
			return fmt.Errorf("password input: %w", err)
		}

		client := api.New(srv)
		token, err := client.Login(login, password)
		if err != nil {
			return fmt.Errorf("login: %w", err)
		}

		if err := session.Save(session.DefaultPath(), &session.Session{
			ServerURL: srv,
			Login:     login,
			Token:     token,
		}); err != nil {
			return fmt.Errorf("save session: %w", err)
		}

		fmt.Printf("logged in as %q\n", login)
		return nil
	},
}
