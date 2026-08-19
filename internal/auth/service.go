package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/dauletsakanayev-lgtm/gophkeeper/internal/model"
	"github.com/dauletsakanayev-lgtm/gophkeeper/internal/storage"
)

// ErrInvalidCredentials - обобщенная ошибка "неверный логин или пароль".
// Возвращается и когда пользователь не найден, и когда пароль не совпал -
// намеренно, чтобы наружу не текло различие (защита от user enumeration).
var ErrInvalidCredentials = errors.New("invalid credentials")

// Service инкапсулирует пользовательскую бизнес-логику аутенфикации:
// регистрация с хешированием пароля, вход с проверкой пароля,
// выдача JWT в обоих случаях.
type Service struct {
	users      storage.UserRepository
	jwt        *JWT
	bcryptCost int
}

// New конструирует Service. cost = 0 означает DefaultCost.
func New(users storage.UserRepository, jwt *JWT, cost int) *Service {
	if cost == 0 {
		cost = DefaultCost
	}
	return &Service{users: users, jwt: jwt, bcryptCost: cost}
}

// Register создает пользователя и сразу выдает JWT- чтобы клиент не делал
// два запроса (register + login). Возвращает storage.ErrUserExists,
// если login уже занят.
func (s *Service) Register(ctx context.Context, login, password string) (token string, user *model.User, err error) {
	hash, err := HashPassword(password, s.bcryptCost)
	if err != nil {
		return "", nil, fmt.Errorf("hash password: %w", err)
	}
	user, err = s.users.Create(ctx, login, hash)
	if err != nil {
		return "", nil, err //ErrUserExists пробрасываем как есть
	}
	token, err = s.jwt.Issue(user.ID, user.Login)
	if err != nil {
		return "", nil, fmt.Errorf("issue token: %w", err)
	}
	return token, user, nil
}

// Login проверяет пару login/password и выдает JWT.
// В любой неудачной ситуации (нет пользователя / пароля не совпал)
// возвращает ErrInvalidCredentials - без указания конкретной причины.
func (s *Service) Login(ctx context.Context, login, password string) (token string, user *model.User, err error) {
	user, err = s.users.GetByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			return "", nil, ErrInvalidCredentials
		}
		return "", nil, fmt.Errorf("get user: %w", err)
	}
	if err := ComparePassword(user.PasswordHash, password); err != nil {
		if errors.Is(err, ErrPasswordMismatch) {
			return "", nil, ErrInvalidCredentials
		}
		return "", nil, fmt.Errorf("compare password: %w", err)
	}
	token, err = s.jwt.Issue(user.ID, user.Login)
	if err != nil {
		return "", nil, fmt.Errorf("issue token: %w", err)
	}
	return token, user, nil
}
