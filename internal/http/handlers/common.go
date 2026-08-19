// Package handlers содержит HTTP-хендлеры GophKeeper.
// Разделены по темам: auth, secrets,sync (по мере добавления).
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog/log"
)

// errorResponse - единый envelope ошибок API: {"error": "..."}
// Клиент всегда парсит одно и то же поле независимо от статуса.
type errorResponse struct {
	Error string `json:"error"`
}

// writeJson сериализует v в JSON и пишет с указанным статусом.
// Ошибка кодирования только логируется - клиент уже получил статус.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Error().Err(err).Msg("encode response")
	}
}

// writeError - короткая обертка для отправки {"error": "..."} с заданным статусом.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorResponse{Error: msg})
}
