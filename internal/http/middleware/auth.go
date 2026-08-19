package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/dauletsakanayev-lgtm/gophkeeper/internal/auth"
)

// ctxKey - приватный тип ключей context, чтобы никто извне не мог их подменить.
type ctxKey int

const (
	// userIDKey - ключ в контексте, под которым хендлер найдет ID
	// аутентифицированного пользователя.
	userIDKey ctxKey = iota
)

// Auth возвращает мидлвар, проверяющий JWT из заголовка Authorization: Bearer <token>.
// При успехе кладет UserID в контекст (см. UserIDFromContext).
// При любой ошибке - 401 без деталей.
func Auth(jwt *auth.JWT) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := r.Header.Get("Authorization")
			const prefix = "Bearer "
			if !strings.HasPrefix(h, prefix) {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			token := strings.TrimSpace(h[len(prefix):])
			claims, err := jwt.Parse(token)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), userIDKey, claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserIDFromContext достает ID пользователя, положенный мидлваром Auth.
// Возвращает (0,false) если запрос не прошел через Auth.
func UserIDFromContext(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(userIDKey).(int64)
	return id, ok
}
