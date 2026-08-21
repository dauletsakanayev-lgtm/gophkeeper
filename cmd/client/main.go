// Command gophkeeper — кросс-платформенный CLI-клиент GophKeeper.
// Реальная логика в internal/client/cli, здесь только запуск.
package main

import (
	"fmt"
	"os"

	"github.com/dauletsakanayev-lgtm/gophkeeper/internal/client/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
