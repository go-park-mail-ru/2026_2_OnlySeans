package filecleanup

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
)

type Storage interface {
	DeleteFile(context.Context, string) error
}

type Worker struct {
	db      *sql.DB
	storage Storage
}

func New(db *sql.DB, storage Storage) *Worker {
	return &Worker{
		db:      db,
		storage: storage,
	}
}

func (w *Worker) ProcessOne(ctx context.Context) (processed bool, operationErr error) {
	tx, err := w.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	var id int64

	defer func() {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
			operationErr = errors.Join(operationErr, fmt.Errorf("rollback file cleanup: %w", rollbackErr))
		}

		if operationErr != nil && id != 0 {
			retryCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Second)
			defer cancel()

			if _, retryErr := w.db.ExecContext(retryCtx, `UPDATE file_cleanup_queue SET attempts=attempts+1,next_attempt_at=now()+interval '1 minute' WHERE file_id=$1`, id); retryErr != nil {
				operationErr = errors.Join(operationErr, fmt.Errorf("schedule cleanup retry for file %d: %w", id, retryErr))
			}
		}
	}()

	err = tx.QueryRowContext(ctx, `SELECT file_id FROM file_cleanup_queue WHERE next_attempt_at<=now() ORDER BY queued_at,file_id LIMIT 1 FOR UPDATE SKIP LOCKED`).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	var key sql.NullString
	if err = tx.QueryRowContext(ctx, `SELECT storage_key FROM file WHERE id=$1 FOR UPDATE`, id).Scan(&key); err != nil {
		return true, err
	}

	var referenced bool
	if err = tx.QueryRowContext(ctx, `SELECT file_has_references($1)`, id).Scan(&referenced); err != nil {
		return true, err
	}

	if referenced {
		if _, err = tx.ExecContext(ctx, `DELETE FROM file_cleanup_queue WHERE file_id=$1`, id); err != nil {
			return true, err
		}
	} else {
		if key.Valid {
			if err = w.storage.DeleteFile(ctx, key.String); err != nil {
				return true, fmt.Errorf("delete detached object: %w", err)
			}
		}

		if _, err = tx.ExecContext(ctx, `DELETE FROM file WHERE id=$1`, id); err != nil {
			return true, err
		}
	}

	return true, tx.Commit()
}

func (w *Worker) Run(ctx context.Context, log logger.Logger) {
	tick := time.NewTicker(5 * time.Second)
	defer tick.Stop()

	for {
		for range 100 {
			if ctx.Err() != nil {
				return
			}

			batchCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			processed, err := w.ProcessOne(batchCtx)

			cancel()

			if err != nil && ctx.Err() == nil && log != nil {
				log.ErrorContext(ctx, "file cleanup failed", map[string]any{"error": err.Error()})
			}

			if !processed || err != nil {
				break
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
	}
}
