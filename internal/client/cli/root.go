package cli

import (
	"github.com/spf13/cobra"
)

const defaultServer = "http://localhost:8081"

var serverURL string

var rootCmd = &cobra.Command{
	Use:           "gophkeeper",
	Short:         "GophKeeper - E2E-шифрованный менеджер паролей",
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&serverURL, "server", "s", "",
		"адрес сервера")

	rootCmd.AddCommand(
		registerCmd,
		loginCmd,
		logoutCmd,
		versionCmd,
		putCmd,
		getCmd,
		listCmd,
		deleteCmd,
	)
}
