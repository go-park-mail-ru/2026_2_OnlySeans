package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/storage"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	createUserQuery = `
		INSERT INTO account (role_id, email, username, password_hash)
		VALUES ((SELECT id FROM role WHERE code = 'reader'), $1, $2, $3)
		RETURNING id
	`

	getUserByEmailQuery = `
		SELECT id, email, username, password_hash,
 (SELECT code FROM role WHERE role.id = account.role_id) AS role_code,
 (SELECT storage_key FROM file WHERE file.id = account.avatar_file_id) AS avatar_filename
		FROM account
		WHERE email = $1 AND deleted_at IS NULL
	`

	getUserByIDQuery = `
		SELECT id, email, username, password_hash,
 (SELECT code FROM role WHERE role.id = account.role_id) AS role_code,
 (SELECT storage_key FROM file WHERE file.id = account.avatar_file_id) AS avatar_filename
		FROM account
		WHERE id = $1 AND deleted_at IS NULL
	`
)

type PostgresUserRepo struct {
	appLogger logger.Logger
	db        *sql.DB
}

func NewPostgresUserRepo(db *sql.DB, appLogger logger.Logger) domain.UserRepository {
	return &PostgresUserRepo{appLogger: appLogger,
		db: db,
	}
}

func (r *PostgresUserRepo) GetUser(ctx context.Context, id domain.UserID) (_ *domain.User, operationErr error) {
	operationLog := storage.StartOperation(ctx, r.appLogger, "postgres", "account.get_user", map[string]any{"user_id": id})
	defer func() { operationLog.Finish(operationErr) }()

	var user domain.User

	operationLog.Stage = "query"
	err := r.db.QueryRowContext(ctx, getUserByIDQuery, id).
		Scan(&user.ID, &user.Email, &user.Username, &user.PasswordHash, &user.Role, &user.AvatarFilename)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("get user %d: %w", id, err)
	}

	return &user, nil
}

func (r *PostgresUserRepo) GetByEmail(ctx context.Context, email string) (_ *domain.User, operationErr error) {
	operationLog := storage.StartOperation(ctx, r.appLogger, "postgres", "account.get_by_email", map[string]any{})
	defer func() { operationLog.Finish(operationErr) }()

	var user domain.User

	operationLog.Stage = "query"
	err := r.db.QueryRowContext(ctx, getUserByEmailQuery, email).
		Scan(&user.ID, &user.Email, &user.Username, &user.PasswordHash, &user.Role, &user.AvatarFilename)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}

	return &user, nil
}

func (r *PostgresUserRepo) Create(ctx context.Context, email, username, passwordHash string) (_ domain.UserID, operationErr error) {
	operationLog := storage.StartOperation(ctx, r.appLogger, "postgres", "account.create", map[string]any{})
	defer func() { operationLog.Finish(operationErr) }()

	var id domain.UserID

	operationLog.Stage = "query"
	err := r.db.QueryRowContext(ctx, createUserQuery, email, username, passwordHash).Scan(&id)
	if err == nil {
		return id, nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == storage.UniqueViolation {
		if pgErr.ConstraintName == "account_username_key" {
			return 0, domain.ErrUsernameTaken
		}
		return 0, domain.ErrUserExists
	}

	return 0, fmt.Errorf("create user: %w", err)
}
