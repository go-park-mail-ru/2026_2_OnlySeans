package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/user/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/user/internal/repository"
	pgutil "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/postgres"
)

const (
	lockAvatarAccountQuery = `
		SELECT id FROM account WHERE id = $1 AND deleted_at IS NULL FOR UPDATE
	`

	insertAvatarFileQuery = `
		INSERT INTO file (storage_key, mime_type, size_bytes, created_by)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`

	attachAvatarQuery = `
		UPDATE account SET avatar_file_id = $1 WHERE id = $2
	`

	getAvatarQuery = `
		SELECT f.storage_key, f.mime_type, f.size_bytes
		FROM account AS a
		LEFT JOIN file AS f ON f.id = a.avatar_file_id
		WHERE a.id = $1 AND a.deleted_at IS NULL
	`

	detachAvatarQuery = `
		UPDATE account SET avatar_file_id = NULL
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id
	`
)

type PostgresAvatarRepo struct {
	db pgutil.DB
}

var _ repository.AvatarRepository = (*PostgresAvatarRepo)(nil)

func NewPostgresAvatarRepo(db pgutil.DB) *PostgresAvatarRepo {
	return &PostgresAvatarRepo{db: db}
}

func (r *PostgresAvatarRepo) SaveAvatar(ctx context.Context, userID int64, file domain.AvatarFile) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin avatar transaction: %w", err)
	}
	defer tx.Rollback()

	var accountID int64
	err = tx.QueryRowContext(ctx, lockAvatarAccountQuery, userID).Scan(&accountID)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrUserNotFound
	}
	if err != nil {
		return fmt.Errorf("lock avatar account: %w", err)
	}

	var fileID int64
	err = tx.QueryRowContext(ctx, insertAvatarFileQuery, file.Filename, file.ContentType, file.Size, accountID).Scan(&fileID)
	if err != nil {
		return fmt.Errorf("insert avatar file: %w", err)
	}

	if _, err := tx.ExecContext(ctx, attachAvatarQuery, fileID, accountID); err != nil {
		return fmt.Errorf("attach avatar to account: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit avatar transaction: %w: %w", domain.ErrCommitUnknown, err)
	}
	return nil
}

func (r *PostgresAvatarRepo) GetAvatar(ctx context.Context, userID int64) (*domain.AvatarFile, error) {
	var filename, contentType sql.NullString
	var size sql.NullInt64
	err := r.db.QueryRowContext(ctx, getAvatarQuery, userID).Scan(&filename, &contentType, &size)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get account avatar: %w", err)
	}
	if !filename.Valid {
		return nil, domain.ErrAvatarNotFound
	}

	return &domain.AvatarFile{
		Filename:    filename.String,
		ContentType: contentType.String,
		Size:        size.Int64,
	}, nil
}

func (r *PostgresAvatarRepo) DeleteAvatar(ctx context.Context, userID int64) error {
	var accountID int64
	err := r.db.QueryRowContext(ctx, detachAvatarQuery, userID).Scan(&accountID)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrUserNotFound
	}
	if err != nil {
		return fmt.Errorf("detach account avatar: %w", err)
	}
	return nil
}
