// Command gophkeeper — кросс-платформенный CLI-клиент GophKeeper.
// Поддерживает регистрацию, аутентификацию, локальное шифрование секретов
// и их синхронизацию с сервером. Реальные подкоманды будут добавлены
// в следующих итерациях (cobra).
package main

import (
	"os"

	"github.com/dauletsakanayev-lgtm/gophkeeper/internal/version"
)

func main() {
	version.Print(os.Stdout, "gophkeeper")
	// TODO(iter5): implement cobra root command with subcommands.
}
