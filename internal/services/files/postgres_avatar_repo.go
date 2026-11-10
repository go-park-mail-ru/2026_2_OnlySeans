package files

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/storage"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

type PostgresAvatarRepo struct {
	log logger.Logger
	db  *sql.DB
}

func NewPostgresAvatarRepo(db *sql.DB, appLogger logger.Logger) (*PostgresAvatarRepo, error) {
	if db == nil {
		return nil, errors.New("avatar database is nil")
	}
	return &PostgresAvatarRepo{log: appLogger, db: db}, nil
}

func (r *PostgresAvatarRepo) SaveAvatar(ctx context.Context, userID domain.UserID, file domain.AvatarFile) (operationErr error) {
	operationLog := storage.StartOperation(ctx, r.log, "postgres", "avatar.save_avatar", map[string]any{"user_id": userID})
	defer func() { operationLog.Finish(operationErr) }()

	operationLog.Stage = "begin"
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin avatar transaction: %w", err)
	}
	defer tx.Rollback()

	var accountID domain.UserID

	operationLog.Stage = "query"
	err = tx.QueryRowContext(ctx, `SELECT id FROM account WHERE id = $1 AND deleted_at IS NULL FOR UPDATE`, userID).Scan(&accountID)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrUserNotFound
	}

	if err != nil {
		return fmt.Errorf("lock avatar account: %w", err)
	}

	var fileID int64

	operationLog.Stage = "query"
	err = tx.QueryRowContext(ctx, `
		INSERT INTO file (storage_key, mime_type, size_bytes, created_by)
		VALUES ($1, $2, $3, $4)
		RETURNING id`, file.Filename, file.ContentType, file.Size, userID).Scan(&fileID)
	if err != nil {
		return fmt.Errorf("insert avatar file: %w", err)
	}

	operationLog.Stage = "query"
	_, err = tx.ExecContext(ctx, `UPDATE account SET avatar_file_id = $1 WHERE id = $2`, fileID, userID)
	if err != nil {
		return fmt.Errorf("set account avatar: %w", err)
	}

	operationLog.Stage = "commit"
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%w: %w", domain.ErrAvatarCommitUnknown, err)
	}

	return nil
}

func (r *PostgresAvatarRepo) GetAvatar(ctx context.Context, userID domain.UserID) (_ *domain.AvatarFile, operationErr error) {
	operationLog := storage.StartOperation(ctx, r.log, "postgres", "avatar.get_avatar", map[string]any{"user_id": userID})
	defer func() { operationLog.Finish(operationErr) }()

	var filename, contentType sql.NullString
	var size sql.NullInt64
	operationLog.Stage = "query"
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

func (r *PostgresAvatarRepo) DeleteAvatar(ctx context.Context, userID domain.UserID) (operationErr error) {
	operationLog := storage.StartOperation(ctx, r.log, "postgres", "avatar.delete_avatar", map[string]any{"user_id": userID})
	defer func() { operationLog.Finish(operationErr) }()

	operationLog.Stage = "begin"
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin avatar deletion: %w", err)
	}
	defer tx.Rollback()

	var accountID domain.UserID

	operationLog.Stage = "query"
	err = tx.QueryRowContext(ctx, `SELECT id FROM account WHERE id = $1 AND deleted_at IS NULL FOR UPDATE`, userID).Scan(&accountID)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrUserNotFound
	}

	if err != nil {
		return fmt.Errorf("lock avatar account: %w", err)
	}

	operationLog.Stage = "query"
	if _, err := tx.ExecContext(ctx, `UPDATE account SET avatar_file_id = NULL WHERE id = $1`, userID); err != nil {
		return fmt.Errorf("detach account avatar: %w", err)
	}

	operationLog.Stage = "commit"
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit avatar deletion: %w", err)
	}

	return nil
}
