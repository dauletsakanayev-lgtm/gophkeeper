// Package api реализует HTTP-клиент к серверу GophKeeper.
// Знает только транспорт: сериализация запросов, добавление JWT в заголовки,
// разбор ответов. Ни шифрования, ни локального кэша — это другие слои.
package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ErrConflict — 409 от сервера (устаревший revision или занятый login).
// ErrUnauthorized — 401 (протух токен или неверные креды).
// ErrNotFound — 404 (нет секрета/юзера).
var (
	ErrConflict     = errors.New("conflict")
	ErrUnauthorized = errors.New("unauthorized")
	ErrNotFound     = errors.New("not found")
)

// Client — тонкий HTTP-клиент к серверу GophKeeper.
// Токен передаётся в конструктор WithToken, чтобы не хранить его в структуре.
type Client struct {
	baseURL string
	http    *http.Client
	token   string
}

// New создаёт клиент без токена (для register/login).
func New(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

// WithToken возвращает клон клиента с установленным JWT-токеном.
// Используется после login — все protected запросы через него.
func (c *Client) WithToken(token string) *Client {
	nc := *c
	nc.token = token
	return &nc
}

// AuthResponse — общий формат ответа /register и /login.
type AuthResponse struct {
	Token string `json:"token"`
}

// Register регистрирует нового пользователя. Возвращает JWT-токен.
func (c *Client) Register(login, password string) (string, error) {
	var resp AuthResponse
	err := c.do(http.MethodPost, "/api/v1/auth/register",
		map[string]string{"login": login, "password": password}, &resp)
	if err != nil {
		return "", err
	}
	return resp.Token, nil
}

// Login аутентифицирует пользователя. Возвращает JWT-токен.
func (c *Client) Login(login, password string) (string, error) {
	var resp AuthResponse
	err := c.do(http.MethodPost, "/api/v1/auth/login",
		map[string]string{"login": login, "password": password}, &resp)
	if err != nil {
		return "", err
	}
	return resp.Token, nil
}

// Secret — представление секрета для HTTP API (совпадает с model.Secret на сервере).
// Data приходит и уходит как base64-строка автоматически через []byte.
type Secret struct {
	ID        string    `json:"id"`
	UserID    int64     `json:"user_id"`
	Type      string    `json:"type"`
	Data      []byte    `json:"data"`
	Meta      string    `json:"meta"`
	Revision  int       `json:"revision"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateSecret создаёт секрет на сервере. Data должна быть уже зашифрована.
func (c *Client) CreateSecret(secretType string, data []byte, meta string) (*Secret, error) {
	body := map[string]any{"type": secretType, "data": data, "meta": meta}
	var resp Secret
	if err := c.do(http.MethodPost, "/api/v1/secrets", body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetSecret возвращает секрет по id.
func (c *Client) GetSecret(id string) (*Secret, error) {
	var resp Secret
	if err := c.do(http.MethodGet, "/api/v1/secrets/"+id, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListSecrets возвращает все секреты пользователя.
func (c *Client) ListSecrets() ([]*Secret, error) {
	var resp []*Secret
	if err := c.do(http.MethodGet, "/api/v1/secrets", nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// UpdateSecret обновляет секрет; revision — версия, которую клиент считает актуальной.
// При несовпадении сервер вернёт ErrConflict + Secret в resp — актуальную серверную копию.
func (c *Client) UpdateSecret(id string, data []byte, meta string, revision int) (*Secret, error) {
	body := map[string]any{"data": data, "meta": meta, "revision": revision}
	var resp Secret
	err := c.do(http.MethodPut, "/api/v1/secrets/"+id, body, &resp)
	// При 409 сервер возвращает актуальный секрет в теле — resp уже заполнен.
	if errors.Is(err, ErrConflict) {
		return &resp, ErrConflict
	}
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// DeleteSecret удаляет секрет.
func (c *Client) DeleteSecret(id string) error {
	return c.do(http.MethodDelete, "/api/v1/secrets/"+id, nil, nil)
}

// do — универсальная точка исполнения HTTP-запросов.
// body: nil / любая JSON-сериализуемая структура; out: nil / указатель на приёмник.
func (c *Client) do(method, path string, body, out any) error {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, c.baseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()

	// Читаем тело один раз — понадобится и для 409, и для ошибок.
	respBody, _ := io.ReadAll(resp.Body)

	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated:
		if out != nil {
			if err := json.Unmarshal(respBody, out); err != nil {
				return fmt.Errorf("decode response: %w", err)
			}
		}
		return nil
	case http.StatusNoContent:
		return nil
	case http.StatusUnauthorized:
		return ErrUnauthorized
	case http.StatusNotFound:
		return ErrNotFound
	case http.StatusConflict:
		// Заполняем out (актуальный секрет от сервера) и возвращаем ErrConflict.
		if out != nil {
			_ = json.Unmarshal(respBody, out)
		}
		return ErrConflict
	default:
		return fmt.Errorf("server returned %d: %s", resp.StatusCode, string(respBody))
	}
}
