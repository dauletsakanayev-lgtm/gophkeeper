// Command gophkeeper-server поднимает HTTP-сервер GophKeeper: принимает
// зашифрованные секреты от клиентов, хранит их в PostgreSQL и раздаёт по
// авторизованному запросу владельца. Реальная логика будет добавлена
// в следующих итерациях.
package main

import (
	"os"

	"github.com/dauletsakanayev-lgtm/gophkeeper/internal/version"
)

func main() {
	version.Print(os.Stdout, "gophkeeper-server")
	// TODO(iter2): parse config, init logger, connect to PostgreSQL, start HTTP server.
}
