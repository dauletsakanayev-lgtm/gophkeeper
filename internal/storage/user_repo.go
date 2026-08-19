package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/dauletsakanayev-lgtm/gophkeeper/internal/model"
	"github.com/jackc/pgx/v5/pgconn"
)

// ErrUserExists - пользователь с таким login уже зарегистрирован.
// ErrUserNotFound - пользователь не найден.
var (
	ErrUserExists   = errors.New("user already exists")
	ErrUserNotFound = errors.New("user not found")
)

// UserRepository - контракт хранилища пользователей.
// Позволяет подменить реализацию в текстах бех mock-фреймворках.
type UserRepository interface {
	Create(ctx context.Context, login, passwordHash string) (*model.User, error)
	GetByLogin(ctx context.Context, login string) (*model.User, error)
}

// PostgreUserRepo - реализация UserRepository поверх PostgreSQL.
type PostgresUserRepo struct {
	db *sql.DB
}

// NewPostgresUserRepo конструирует репозиторий, принимая уже открытое соединение.
func NewPostgresUserRepo(db *sql.DB) *PostgresUserRepo {
	return &PostgresUserRepo{db: db}
}

// Create сохраняет нового пользователя. Возвращает ErrUserExists,
// если login занят (нарушение UNIQUE constraint PostgreSQL).
func (r *PostgresUserRepo) Create(ctx context.Context, login, passwordHash string) (*model.User, error) {
	const q = `
		INSERT INTO users(login, password_hash)
		VALUES ($1,$2)
		RETURNING id, login, password_hash, created_at
	`
	u := &model.User{}
	err := r.db.QueryRowContext(ctx, q, login, passwordHash).
		Scan(&u.ID, &u.Login, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { //unique_violation
			return nil, ErrUserExists
		}
		return nil, fmt.Errorf("insert user: %w", err)
	}
	return u, nil
}

// GetByLogin возвращает пользователя по login или ErrUserNotFound.
func (r *PostgresUserRepo) GetByLogin(ctx context.Context, login string) (*model.User, error) {
	const q = `
		SELECT id, login, password_hash, created_at
		FROM users
		WHERE login = $1
	`
	u := &model.User{}
	err := r.db.QueryRowContext(ctx, q, login).
		Scan(&u.ID, &u.Login, &u.PasswordHash, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("select user: %w", err)
	}
	return u, nil
}
