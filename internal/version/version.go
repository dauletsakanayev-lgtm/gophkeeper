// Package version хранит сведения о сборке (версия, коммит, дата),
// которые проставляются в -ldflags при компиляции. Значения по умолчанию
// используются при запуске неpaзмеченного бинарника (например, `go run`).
//
// Значения выставляются Makefile'ом через:
//
//	-X 'github.com/dauletsakanayev-lgtm/gophkeeper/internal/version.Version=...'
//	-X 'github.com/dauletsakanayev-lgtm/gophkeeper/internal/version.Commit=...'
//	-X 'github.com/dauletsakanayev-lgtm/gophkeeper/internal/version.Date=...'
package version

import (
	"fmt"
	"io"
)

// Version — семантическая версия бинарника или "dev" при разработке.
// Commit — короткий SHA1 текущего git-коммита.
// Date — метка времени сборки в формате RFC 3339 UTC.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// Print выводит сведения о сборке в человекочитаемом формате.
// Используется бинарями сервера и клиента при старте (и в клиенте — по команде version).
func Print(w io.Writer, appName string) {
	fmt.Fprintf(w, "%s\n  version: %s\n  commit:  %s\n  built:   %s\n",
		appName, Version, Commit, Date)
}
