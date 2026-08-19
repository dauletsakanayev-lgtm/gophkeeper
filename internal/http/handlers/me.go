package handlers

import (
	"net/http"

	"github.com/dauletsakanayev-lgtm/gophkeeper/internal/http/middleware"
)

// Me возвращает user_id аутентифицированного пользователя из JWT.
// Используется для быстрой проверки, что токен работает.
func Me() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := middleware.UserIDFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "inauthorized")
			return
		}
		writeJSON(w, http.StatusOK, map[string]int64{"user_id": id})
	}
}
