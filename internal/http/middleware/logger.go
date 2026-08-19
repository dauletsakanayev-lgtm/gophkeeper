// Package middleware содержит HTTP - мидлвары GophKeeper: логирование запросов
// и защиту энпоинтов через JWT.
package middleware

import (
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

// statusRecorder оборачивает ResponseWriter, чтобы зафиксировать реальный статус
// ответа - стандартный http.ResponseWriter об этом не сообщает.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

// Logger логирует метод, путь, статус ответа и длительности обработки каждого запроса.
// При stasus >= 500 - ERROR, при 400..499 - WARN, иначе INFO.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		ev := log.Info()
		switch {
		case rec.status >= 500:
			ev = log.Error()
		case rec.status >= 400:
			ev = log.Warn()
		}
		ev.Str("method", r.Method).Str("path", r.URL.Path).
			Int("status", rec.status).Dur("duration", time.Since(start)).
			Msg("http request")
	})
}
