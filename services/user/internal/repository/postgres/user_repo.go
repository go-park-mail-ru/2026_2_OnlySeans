package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/user/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/user/internal/repository"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
	pgutil "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/postgres"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/storage"
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

	updateProfileQuery = `
		UPDATE account
		SET email = COALESCE($2, email),
		    username = COALESCE($3, username)
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, email, username, password_hash,
		(SELECT code FROM role WHERE role.id = account.role_id),
		(SELECT storage_key FROM file WHERE file.id = account.avatar_file_id)
	`
)

type PostgresUserRepo struct {
	appLogger logger.Logger
	db        pgutil.DB
}

func NewPostgresUserRepo(db pgutil.DB, appLogger logger.Logger) repository.UserRepository {
	return &PostgresUserRepo{
		appLogger: appLogger,
		db:        db,
	}
}

func (r *PostgresUserRepo) CreateUser(ctx context.Context, user domain.User) (_ int64, operationErr error) {
	operationLog := storage.StartOperation(ctx, r.appLogger, "postgres", "account.create", map[string]any{})
	defer func() { finishUserOperation(operationLog, operationErr) }()

	var id int64

	operationLog.Stage = "query"
	err := r.db.QueryRowContext(ctx, createUserQuery, user.Email, user.Username, user.PasswordHash).Scan(&id)
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

func (r *PostgresUserRepo) GetUserByEmail(ctx context.Context, email string) (_ domain.User, operationErr error) {
	operationLog := storage.StartOperation(ctx, r.appLogger, "postgres", "account.get_by_email", map[string]any{})
	defer func() { finishUserOperation(operationLog, operationErr) }()

	var user domain.User

	operationLog.Stage = "query"
	err := r.db.QueryRowContext(ctx, getUserByEmailQuery, email).
		Scan(&user.ID, &user.Email, &user.Username, &user.PasswordHash, &user.Role, &user.AvatarFilename)

	if errors.Is(err, sql.ErrNoRows) {
		return domain.User{}, domain.ErrUserNotFound
	}

	if err != nil {
		return domain.User{}, fmt.Errorf("get user by email: %w", err)
	}

	return user, nil
}

func (r *PostgresUserRepo) GetUserByID(ctx context.Context, id int64) (_ domain.User, operationErr error) {
	operationLog := storage.StartOperation(ctx, r.appLogger, "postgres", "account.get_user", map[string]any{"user_id": id})
	defer func() { finishUserOperation(operationLog, operationErr) }()

	var user domain.User

	operationLog.Stage = "query"
	err := r.db.QueryRowContext(ctx, getUserByIDQuery, id).
		Scan(&user.ID, &user.Email, &user.Username, &user.PasswordHash, &user.Role, &user.AvatarFilename)

	if errors.Is(err, sql.ErrNoRows) {
		return domain.User{}, domain.ErrUserNotFound
	}

	if err != nil {
		return domain.User{}, fmt.Errorf("get user %d: %w", id, err)
	}

	return user, nil
}

func (r *PostgresUserRepo) UpdateProfile(ctx context.Context, userID int64, username, email *string) (_ domain.User, operationErr error) {
	operationLog := storage.StartOperation(ctx, r.appLogger, "postgres", "profile.update_profile", map[string]any{"user_id": userID})
	defer func() { finishUserOperation(operationLog, operationErr) }()

	var user domain.User

	operationLog.Stage = "query"
	err := r.db.QueryRowContext(ctx, updateProfileQuery, userID, email, username).
		Scan(&user.ID, &user.Email, &user.Username, &user.PasswordHash, &user.Role, &user.AvatarFilename)

	if errors.Is(err, sql.ErrNoRows) {
		return domain.User{}, domain.ErrUserNotFound
	}

	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) && pgErr.Code == storage.UniqueViolation {
		switch pgErr.ConstraintName {
		case "account_username_key":
			return domain.User{}, domain.ErrUsernameTaken
		case "account_email_key":
			return domain.User{}, domain.ErrUserExists
		}
	}

	if err != nil {
		return domain.User{}, err
	}

	return user, nil
}

func finishUserOperation(operation *storage.Operation, err error) {
	operation.NotFound = errors.Is(err, sql.ErrNoRows) || errors.Is(err, domain.ErrUserNotFound)
	operation.Rejected = errors.Is(err, domain.ErrUserExists) || errors.Is(err, domain.ErrUsernameTaken)
	operation.Finish(err)
}
