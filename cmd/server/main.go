// Command gophkeeper-server поднимает HTTP-сервер GophKeeper: принимает
// зашифрованные секреты от клиентов, хранит их в PostgreSQL и раздаёт по
// авторизованному запросу владельца.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/dauletsakanayev-lgtm/gophkeeper/internal/auth"
	httpsrv "github.com/dauletsakanayev-lgtm/gophkeeper/internal/http"
	"github.com/dauletsakanayev-lgtm/gophkeeper/internal/storage"
	"github.com/dauletsakanayev-lgtm/gophkeeper/internal/version"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	version.Print(os.Stdout, "gophkeeper-server")

	cfg, err := parseCongif()
	if err != nil {
		log.Fatal().Err(err).Msg("ошибка конфигурации")
	}
	setLogLevel(cfg.LogLevel)

	db, err := storage.Open(cfg.DatabaseDNS)
	if err != nil {
		log.Fatal().Err(err).Msg("подключение к PostgreSQL")
	}
	defer db.Close()

	if err := storage.Migrate(db); err != nil {
		log.Fatal().Err(err).Msg("применение миграции")
	}
	log.Info().Msg("миграции применены")

	userRepo := storage.NewPostgresUserRepo(db)
	secretRepo := storage.NewPostgresSecretRepo(db)
	jwtIssuer := auth.NewJWT(cfg.JWTSecret, cfg.JWTTTL)
	authSvc := auth.New(userRepo, jwtIssuer, auth.DefaultCost)

	ctx, stop := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	srv := httpsrv.New(cfg.Addr, authSvc, jwtIssuer, secretRepo)
	if err := srv.Run(ctx); err != nil {
		log.Fatal().Err(err).Msg("HTTP-сервер завершился с ошибкой")
	}
}

// setLogLevel настраивает глобальный уровень zerolog из строки когфиги.
// Неизвестое значение > InfoLevel.
func setLogLevel(level string) {
	lvl, err := zerolog.ParseLevel(level)
	if err != nil || lvl == zerolog.NoLevel {
		lvl = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(lvl)
}
