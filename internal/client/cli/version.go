package cli

import (
	"os"

	"github.com/dauletsakanayev-lgtm/gophkeeper/internal/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Вывести версию и дату сборки",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		version.Print(os.Stdout, "gophkeeper")
	},
}
