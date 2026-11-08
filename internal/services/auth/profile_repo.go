package auth

import (
	"context"
	"database/sql"
	"errors"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/jackc/pgx/v5/pgconn"
)

type ProfileRepo struct{ db *sql.DB }

func NewProfileRepo(db *sql.DB) *ProfileRepo { return &ProfileRepo{db: db} }

func (r *ProfileRepo) UpdateProfile(ctx context.Context, id domain.UserID, email, username string) (*domain.User, error) {
	var user domain.User
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
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		if pgErr.ConstraintName == "account_username_key" {
			return nil, domain.ErrUsernameTaken
		}
		return nil, domain.ErrUserExists
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}
