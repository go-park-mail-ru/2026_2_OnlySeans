package auth

import (
	"context"
	"database/sql"
	"errors"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/storage"
	"github.com/jackc/pgx/v5/pgconn"
)

func (r *PostgresUserRepo) UpdateProfile(ctx context.Context, id domain.UserID, email, username string) (_ *domain.User, operationErr error) {
	operationLog := storage.StartOperation(ctx, r.log, "postgres", "profile.update_profile", map[string]any{"user_id": id})
	defer func() { operationLog.Finish(operationErr) }()

	var user domain.User
	operationLog.Stage = "query"
	err := r.db.QueryRowContext(ctx, `UPDATE account SET email=$2,username=$3
		WHERE id=$1 AND deleted_at IS NULL
		RETURNING id,email,username,password_hash,
		(SELECT code FROM role WHERE role.id=account.role_id),
		(SELECT storage_key FROM file WHERE file.id=account.avatar_file_id)`, id, email, username).
		Scan(&user.ID, &user.Email, &user.Username, &user.PasswordHash, &user.Role, &user.AvatarFilename)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == storage.UniqueViolation {
		switch pgErr.ConstraintName {
		case "account_username_key":
			return nil, domain.ErrUsernameTaken
		case "account_email_key":
			return nil, domain.ErrUserExists
		}
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}
