package cli

import (
	"fmt"

	"github.com/dauletsakanayev-lgtm/gophkeeper/internal/client/session"
	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Завершить сессию: удалить локальный JWT-токен",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := session.Delete(session.DefaultPath()); err != nil {
			return fmt.Errorf("logout: %w", err)
		}
		fmt.Println("logged out")
		return nil
	},
}
