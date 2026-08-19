package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims -payload JWT.
// UserID кладем как отдельное поле, чтобы middleware могло дешево прочитать
// его из токена без дополнительного запроса к БД.
type Claims struct {
	UserID int64  `json:"user_id"`
	Login  string `json:"login"`
	jwt.RegisteredClaims
}

// ErrInvalidToken - общее сообщение при любой проблеме с токеном
// (подпись, срок. формат). Наружу отдаем без деталей.
var ErrInvalidToken = errors.New("invalid token")

// JWT выпускает и разбирает HS256-токены с одним симметричным секретом.
// Экземпляр безопасен для паралельного использования.
type JWT struct {
	secret []byte
	ttl    time.Duration
}

// NewJWT конструирует issuer/parser. TTL используется только при Issue,
// Parse проверяет exp из самого токена.
func NewJWT(secret string, ttl time.Duration) *JWT {
	return &JWT{secret: []byte(secret), ttl: ttl}
}

// Issue выпускает подписанный JWT для указанного пользователя.
// Устанавливает Subject = fmt(userID), ExpiresAt = now+TTL, IssuedAt = now.
func (j *JWT) Issue(userID int64, login string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		Login:  login,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("%d", userID),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(j.ttl)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := tok.SignedString(j.secret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return s, nil
}

// Parse проверяет подпись, срок действия и алгоритм. При любой неудаче
// возвращает ErrInvalidToken - детали в ответ клиенту уходит не должны.
func (j *JWT) Parse(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	tok, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return j.secret, nil
	})
	if err != nil || !tok.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
