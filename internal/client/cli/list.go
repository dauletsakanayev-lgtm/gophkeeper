package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Показать список секретов (id / type / meta / updated_at)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := mustSession()
		if err != nil {
			return err
		}
		items, err := authClient(s).ListSecrets()
		if err != nil {
			return fmt.Errorf("list: %w", err)
		}
		if len(items) == 0 {
			fmt.Println("no secrets")
			return nil
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tTYPE\tMETA\tUPDATED")
		for _, it := range items {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
				it.ID, it.Type, it.Meta, it.UpdatedAt.Format("2006-01-02 15:04:05"))
		}
		return w.Flush()
	},
}
