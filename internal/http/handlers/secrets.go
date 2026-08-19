package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/dauletsakanayev-lgtm/gophkeeper/internal/http/middleware"
	"github.com/dauletsakanayev-lgtm/gophkeeper/internal/model"
	"github.com/dauletsakanayev-lgtm/gophkeeper/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// createSecretRequest — тело запроса POST /secrets.
// Data передаётся байтами (в JSON — автоматически base64 через Go stdlib).
type createSecretRequest struct {
	Type string `json:"type"`
	Data []byte `json:"data"`
	Meta string `json:"meta"`
}

// updateSecretRequest — тело запроса PUT /secrets/{id}.
// Revision — версия секрета, которую клиент считает актуальной.
// Сервер сравнивает её с текущей: не совпало → 409.
type updateSecretRequest struct {
	Data     []byte `json:"data"`
	Meta     string `json:"meta"`
	Revision int    `json:"revision"`
}

// CreateSecret возвращает handler, сохраняющий новый секрет пользователя.
// user_id берётся из JWT (context), body — тип/данные/мета.
// Успех: 201 Created + сам секрет (с сгенерированным id, revision=1).
func CreateSecret(repo storage.SecretRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.UserIDFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		var req createSecretRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json")
			return
		}
		if req.Type == "" || len(req.Data) == 0 {
			writeError(w, http.StatusBadRequest, "type and data are required")
			return
		}

		s, err := repo.Create(r.Context(), userID, req.Type, req.Data, req.Meta)
		if err != nil {
			log.Error().Err(err).Str("op", "secrets.create").Msg("internal error")
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		writeJSON(w, http.StatusCreated, s)
	}
}

// GetSecret возвращает handler, отдающий один секрет пользователя по id (UUID).
// Успех: 200 + секрет. 404 — если такого секрета нет у этого пользователя.
func GetSecret(repo storage.SecretRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.UserIDFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		id := chi.URLParam(r, "id")

		s, err := repo.Get(r.Context(), userID, id)
		switch {
		case errors.Is(err, storage.ErrSecretNotFound):
			writeError(w, http.StatusNotFound, "secret not found")
			return
		case err != nil:
			log.Error().Err(err).Str("op", "secrets.get").Msg("internal error")
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		writeJSON(w, http.StatusOK, s)
	}
}

// ListSecrets возвращает handler, отдающий все секреты пользователя.
// Ответ — массив, отсортированный по updated_at DESC (свежие сверху).
// Успех: 200 + [] (или пустой массив, если секретов ещё нет).
func ListSecrets(repo storage.SecretRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.UserIDFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		items, err := repo.List(r.Context(), userID)
		if err != nil {
			log.Error().Err(err).Str("op", "secrets.list").Msg("internal error")
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		if items == nil {
			items = make([]*model.Secret, 0) // безопасный пустой массив в JSON
		}
		writeJSON(w, http.StatusOK, items)
	}
}

// UpdateSecret возвращает handler, обновляющий data/meta существующего секрета.
// Клиент шлёт revision (свою версию); при несовпадении с серверной — 409.
// Успех: 200 + обновлённый секрет (с revision+1, свежим updated_at).
func UpdateSecret(repo storage.SecretRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.UserIDFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		id := chi.URLParam(r, "id")

		var req updateSecretRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json")
			return
		}
		if len(req.Data) == 0 {
			writeError(w, http.StatusBadRequest, "data is required")
			return
		}
		if req.Revision <= 0 {
			writeError(w, http.StatusBadRequest, "revision must be positive")
			return
		}

		s, err := repo.Update(r.Context(), userID, id, req.Revision, req.Data, req.Meta)
		switch {
		case errors.Is(err, storage.ErrSecretNotFound):
			writeError(w, http.StatusNotFound, "secret not found")
			return
		case errors.Is(err, storage.ErrRevisionMismatch):
			// Возвращаем актуальный секрет с сервера, чтобы клиент сразу
			// увидел свежее состояние и разрулил конфликт.
			current, gerr := repo.Get(r.Context(), userID, id)
			if gerr != nil {
				log.Error().Err(gerr).Str("op", "secrets.update.get_after_conflict").Msg("internal error")
				writeError(w, http.StatusConflict, "revision mismatch")
				return
			}
			writeJSON(w, http.StatusConflict, current)
			return
		case err != nil:
			log.Error().Err(err).Str("op", "secrets.update").Msg("internal error")
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		writeJSON(w, http.StatusOK, s)
	}
}

// DeleteSecret возвращает handler, удаляющий секрет пользователя.
// Успех: 204 No Content. 404 — если секрета нет.
func DeleteSecret(repo storage.SecretRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.UserIDFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		id := chi.URLParam(r, "id")

		err := repo.Delete(r.Context(), userID, id)
		switch {
		case errors.Is(err, storage.ErrSecretNotFound):
			writeError(w, http.StatusNotFound, "secret not found")
			return
		case err != nil:
			log.Error().Err(err).Str("op", "secrets.delete").Msg("internal error")
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
