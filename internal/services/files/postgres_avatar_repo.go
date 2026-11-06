package files

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

type PostgresAvatarRepo struct {
	db *sql.DB
}

func NewPostgresAvatarRepo(db *sql.DB) (*PostgresAvatarRepo, error) {
	if db == nil {
		return nil, errors.New("avatar database is nil")
	}
	return &PostgresAvatarRepo{db: db}, nil
}

func (r *PostgresAvatarRepo) SaveAvatar(ctx context.Context, userID domain.UserID, file domain.AvatarFile) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin avatar transaction: %w", err)
	}
	defer tx.Rollback()

	var accountID domain.UserID
	err = tx.QueryRowContext(ctx, `SELECT id FROM account WHERE id = $1 AND deleted_at IS NULL FOR UPDATE`, userID).Scan(&accountID)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrUserNotFound
	}
	if err != nil {
		return fmt.Errorf("lock avatar account: %w", err)
	}

	var fileID int64
	err = tx.QueryRowContext(ctx, `
		INSERT INTO file (storage_key, mime_type, size_bytes, created_by)
		VALUES ($1, $2, $3, $4)
		RETURNING id`, file.Filename, file.ContentType, file.Size, userID).Scan(&fileID)
	if err != nil {
		return fmt.Errorf("insert avatar file: %w", err)
	}
	_, err = tx.ExecContext(ctx, `UPDATE account SET avatar_file_id = $1 WHERE id = $2`, fileID, userID)
	if err != nil {
		return fmt.Errorf("set account avatar: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%w: %w", domain.ErrAvatarCommitUnknown, err)
	}
	return nil
}

var _ domain.AvatarRepository = (*PostgresAvatarRepo)(nil)

func (r *PostgresAvatarRepo) GetAvatar(ctx context.Context, userID domain.UserID) (*domain.AvatarFile, error) {
	var filename, contentType sql.NullString
	var size sql.NullInt64
	err := r.db.QueryRowContext(ctx, `
  SELECT f.storage_key, f.mime_type, f.size_bytes
  FROM account AS a
  LEFT JOIN file AS f ON f.id = a.avatar_file_id
  WHERE a.id = $1 AND a.deleted_at IS NULL
 `, userID).Scan(&filename, &contentType, &size)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get account avatar: %w", err)
	}
	if !filename.Valid {
		return nil, domain.ErrFileNotFound
	}
	return &domain.AvatarFile{Filename: filename.String, ContentType: contentType.String, Size: size.Int64}, nil
}

func (r *PostgresAvatarRepo) DeleteAvatar(ctx context.Context, userID domain.UserID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin avatar deletion: %w", err)
	}
	defer tx.Rollback()
	var accountID domain.UserID
	err = tx.QueryRowContext(ctx, `SELECT id FROM account WHERE id = $1 AND deleted_at IS NULL FOR UPDATE`, userID).Scan(&accountID)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrUserNotFound
	}
	if err != nil {
		return fmt.Errorf("lock avatar account: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE account SET avatar_file_id = NULL WHERE id = $1`, userID); err != nil {
		return fmt.Errorf("detach account avatar: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit avatar deletion: %w", err)
	}
	return nil
}
