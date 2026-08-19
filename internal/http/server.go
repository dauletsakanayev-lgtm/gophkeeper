// Package http собирает HTTP-сервер GophKeeper: роутинг chi, мидлвары
// и graceful shutdown.
package http

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/dauletsakanayev-lgtm/gophkeeper/internal/auth"
	"github.com/dauletsakanayev-lgtm/gophkeeper/internal/http/handlers"
	"github.com/dauletsakanayev-lgtm/gophkeeper/internal/http/middleware"
	"github.com/dauletsakanayev-lgtm/gophkeeper/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// Server - HTTP - сервер приложения. Держит адрес и уже собранный handler.
type Server struct {
	addr    string
	handler http.Handler
}

// New собирает роутер chi с эндпоинтами auth (register/login) и
// служебным /api/v1/me под JWT-guard'ом для проверки токена.
func New(addr string, authSvc *auth.Service, jwt *auth.JWT, secretRepo storage.SecretRepository) *Server {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/register", handlers.Register(authSvc))
		r.Post("/auth/login", handlers.Login(authSvc))

		// Защищённая группа — доступна только с валидным JWT.
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(jwt))
			r.Get("/me", handlers.Me())

			r.Post("/secrets", handlers.CreateSecret(secretRepo))
			r.Get("/secrets", handlers.ListSecrets(secretRepo))
			r.Get("/secrets/{id}", handlers.GetSecret(secretRepo))
			r.Put("/secrets/{id}", handlers.UpdateSecret(secretRepo))
			r.Delete("/secrets/{id}", handlers.DeleteSecret(secretRepo))
		})
	})

	return &Server{addr: addr, handler: r}
}

// Run запускает HTTP- сервер и блокируется до отмены ctx. При отмене выполняет
// graceful shutdown с таймаутом 30 секунд. Возвращает ошибку от ListAndServe
// (кроме штатного ErrServerClosed) или ошибку Shutdown.
func (s *Server) Run(ctx context.Context) error {
	srv := &http.Server{
		Addr:         s.addr,
		Handler:      s.handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	serveErr := make(chan error, 1)
	go func() {
		log.Info().Str("Addr", s.addr).Msg("HTTP-сервер запущен")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serveErr <- err
			return
		}
		serveErr <- nil
	}()

	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
	}

	log.Info().Msg("останавливаем HTTP-сервер")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("http shutdown: %w", err)
	}
	log.Info().Msg("HTTP-сервер завершил работу корректно")
	return nil
}
