package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/dauletsakanayev-lgtm/gophkeeper/internal/auth"
	"github.com/dauletsakanayev-lgtm/gophkeeper/internal/storage"
	"github.com/rs/zerolog/log"
)

// authService - минимальный контракт для тестируемости хендлеров.
// В проде это *auth.Service, в тестах - заглушка.
/*type authService interface {
	Register(ctx context.Context, login, password string) (token string, user *struct{ ID int64 }, err error)
	Login(ctx context.Context, login, password string) (token string, user *struct{ ID int64 }, err error)
}*/

// authRequest - тело запроса /register и /login.
type authReguest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// authResponse - успешный ответ /register и /login.
type authResponse struct {
	Token string `json:"token"`
}

// Register возвращает http.HandlerFunc, регистрирующего нового пользователя.
// Успех: 200 OK + {token}. Заголовок Authorization: Bearer <token> для удобства.
// Ошибка: 400 (битый JSON  или пустые поля), 409 (login занят), 500 (внутренняя).
func Register(svc *auth.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req authReguest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json")
			return
		}
		if req.Login == "" || req.Password == "" {
			writeError(w, http.StatusBadRequest, "login and password are required")
			return
		}

		token, _, err := svc.Register(r.Context(), req.Login, req.Password)
		switch {
		case errors.Is(err, storage.ErrUserExists):
			writeError(w, http.StatusConflict, "login already taken")
			return
		case err != nil:
			log.Error().Err(err).Str("op", "register").Msg("internal error")
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		w.Header().Set("Authorization", "Bearer "+token)
		writeJSON(w, http.StatusOK, authResponse{Token: token})
	}
}

// Login возвращает http.HandlerFunc, аутентифицируещего существующего пользователя.
// Успех: 200 OK + {token}. Ошибки: 400 (битый JSON или пустые поля),
// 401 (неверный логин или пароль), 500 (внутренняя).
func Login(svc *auth.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req authReguest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid hson")
			return
		}
		if req.Login == "" || req.Password == "" {
			writeError(w, http.StatusBadRequest, "login and password are required")
		}
		token, _, err := svc.Login(r.Context(), req.Login, req.Password)
		switch {
		case errors.Is(err, auth.ErrInvalidCredentials):
			writeError(w, http.StatusUnauthorized, "invalid credentials")
			return
		case err != nil:
			log.Error().Err(err).Str("op", "login").Msg("internal error")
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		w.Header().Set("Authorization", "Bearer "+token)
		writeJSON(w, http.StatusOK, authResponse{Token: token})
	}
}
