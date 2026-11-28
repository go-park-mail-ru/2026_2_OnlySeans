package onlineconf

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	oc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain/onlineconf"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/storage"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

const (
	restrictViolation = "23001"
	lockStateQuery    = `SELECT revision FROM oc_state WHERE id = 1 FOR UPDATE`
	actorNameQuery    = `SELECT username FROM account WHERE id = $1 AND deleted_at IS NULL`
	scopeIDQuery      = `SELECT id FROM oc_scopes WHERE name = $1`
	storageBackend    = "postgres"
	mutationOperation = "onlineconf.mutate"
)

type PostgresRepository struct {
	db        *sql.DB
	appLogger logger.Logger
}

type mutation struct {
	tx        *sql.Tx
	actorID   domain.UserID
	actorName string
	revision  int64
}

type rowScanner interface{ Scan(...any) error }

var _ oc.Repository = (*PostgresRepository)(nil)

func NewPostgresRepository(db *sql.DB, appLogger logger.Logger) oc.Repository {
	return &PostgresRepository{db: db, appLogger: appLogger}
}

func (r *PostgresRepository) mutate(ctx context.Context, actorID domain.UserID, change func(*mutation) error) (operationErr error) {
	operation := storage.StartOperation(ctx, r.appLogger, storageBackend, mutationOperation, nil)
	defer func() { operation.Finish(operationErr) }()

	if actorID < oc.MinUserID || int64(actorID) > oc.MaxSafeID {
		return oc.ErrInvalidUserID
	}

	operation.Stage = "begin"

	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return fmt.Errorf("begin onlineconf mutation: %w", err)
	}

	defer rollbackTransaction(tx, &operationErr)
	m := &mutation{tx: tx, actorID: actorID}
	operation.Stage = "lock"

	if err := tx.QueryRowContext(ctx, lockStateQuery).Scan(&m.revision); err != nil {
		return fmt.Errorf("lock onlineconf state: %w", err)
	}

	m.revision++
	if err := tx.QueryRowContext(ctx, actorNameQuery, actorID).Scan(&m.actorName); err != nil {
		return repositoryError(err)
	}

	operation.Stage = "query"

	if err := change(m); err != nil {
		return repositoryError(err)
	}

	operation.Stage = "snapshot"

	if err := m.saveSnapshot(ctx); err != nil {
		return repositoryError(err)
	}

	operation.Stage = "commit"

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%w: %w", domain.ErrCommitUnknown, err)
	}

	return nil
}

func readTransaction[T any](ctx context.Context, db *sql.DB, read func(*sql.Tx) (T, error)) (_ T, operationErr error) {
	var zero T

	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return zero, fmt.Errorf("begin onlineconf read: %w", err)
	}

	defer rollbackTransaction(tx, &operationErr)

	value, err := read(tx)
	if err != nil {
		return zero, repositoryError(err)
	}

	if err := tx.Commit(); err != nil {
		return zero, fmt.Errorf("commit onlineconf read: %w", err)
	}

	return value, nil
}

func repositoryError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return oc.ErrNotFound
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case storage.UniqueViolation:
			return fmt.Errorf("%w: %w", oc.ErrDuplicateKey, err)
		case storage.ForeignKeyViolation, restrictViolation:
			return fmt.Errorf("%w: %w", oc.ErrResourceInUse, err)
		}
	}

	return err
}

func validateResource(id, version int64) error {
	if id < oc.MinUserID || id > oc.MaxSafeID {
		return oc.ErrInvalidID
	}

	if version < 1 {
		return oc.ErrInvalidVersion
	}

	return nil
}

func checkVersion(expected, actual int64) error {
	if expected != actual {
		return fmt.Errorf("%w: expected %d, actual %d", oc.ErrVersionConflict, expected, actual)
	}

	return nil
}

func listOptions(options oc.ListOptions) (oc.ListOptions, error) {
	if options.Limit == 0 {
		options.Limit = oc.DefaultListLimit
	}

	if options.Limit < 0 || options.Limit > oc.MaxListLimit || options.Offset < 0 {
		return options, oc.ErrInvalidPagination
	}

	return options, nil
}

func scopeID(ctx context.Context, tx *sql.Tx, name string) (int64, error) {
	var id int64
	err := tx.QueryRowContext(ctx, scopeIDQuery, name).Scan(&id)

	return id, repositoryError(err)
}

func collectRows[T any](rows *sql.Rows, scan func(rowScanner) (T, error)) (_ []T, operationErr error) {
	defer func() {
		if err := rows.Close(); err != nil {
			operationErr = errors.Join(operationErr, err)
		}
	}()

	values := make([]T, 0)

	for rows.Next() {
		value, err := scan(rows)
		if err != nil {
			return nil, err
		}

		values = append(values, value)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return values, nil
}

func rollbackTransaction(tx *sql.Tx, operationErr *error) {
	if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
		*operationErr = errors.Join(*operationErr, err)
	}
}

type page[T any] struct {
	items []T
	total int
}

func listPage[T any](ctx context.Context, db *sql.DB, countQuery, listQuery string, args []any, limit, offset int, scan func(rowScanner) (T, error)) ([]T, int, error) {
	result, err := readTransaction(ctx, db, func(tx *sql.Tx) (page[T], error) {
		var result page[T]
		if err := tx.QueryRowContext(ctx, countQuery, args...).Scan(&result.total); err != nil {
			return result, err
		}

		queryArgs := append(append([]any{}, args...), limit, offset)

		rows, err := tx.QueryContext(ctx, listQuery, queryArgs...)
		if err != nil {
			return result, err
		}

		result.items, err = collectRows(rows, scan)

		return result, err
	})

	return result.items, result.total, err
}
