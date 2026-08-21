package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Удалить секрет по id",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		s, err := mustSession()
		if err != nil {
			return err
		}
		if err := authClient(s).DeleteSecret(id); err != nil {
			return fmt.Errorf("delete: %w", err)
		}
		fmt.Printf("deleted %s\n", id)
		return nil
	},
}
