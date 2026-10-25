package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
)

const (
	//регистрация
	createUserQuery = `
			INSERT INTO account (role_id, email, username, password_hash)
			VALUES ((SELECT id FROM role WHERE code = 'reader'), $1, $2, $3)
			RETURNING id
		`

	//вход
	getUserByEmailQuery = `
			SELECT id, email, username, password_hash
			FROM account
			WHERE email = $1 AND deleted_at IS NULL
		`

	//аворизация
	getUserByIDQuery = `
			SELECT id, email, username, password_hash
			FROM account
			WHERE id = $1 AND deleted_at IS NULL
		`

	uniqueViolation = "23505" //код когда нарушено ограничение
)

type PostgresUserRepo struct {
	db *sql.DB //пул соединений
}

func NewPostgresUserRepo(db *sql.DB) *PostgresUserRepo {
	return &PostgresUserRepo{db: db}
}

var _ UserRepository = (*PostgresUserRepo)(nil) //проверка что подходит под интерфейс

// после входа
func (r *PostgresUserRepo) GetUser(ctx context.Context, id UserID) (*User, error) {
	var user User
	err := r.db.QueryRowContext(ctx, getUserByIDQuery, id).Scan(&user.ID, &user.Email, &user.Username, &user.PasswordHash)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user %d: %w", id, err)
	}
	return &user, nil
}

// при входе
func (r *PostgresUserRepo) GetByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	err := r.db.QueryRowContext(ctx, getUserByEmailQuery, email).Scan(&user.ID, &user.Email, &user.Username, &user.PasswordHash)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return &user, nil
}

// регистрация
func (r *PostgresUserRepo) Create(ctx context.Context, email, username, passwordHash string) (UserID, error) {
	var id UserID
	err := r.db.QueryRowContext(ctx, createUserQuery, email, username, passwordHash).Scan(&id)
	if err == nil {
		return id, nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
		if pgErr.ConstraintName == "account_username_key" {
			return 0, ErrUsernameTaken
		}
		return 0, ErrUserExists
	}
	return 0, fmt.Errorf("create user: %w", err)
}
