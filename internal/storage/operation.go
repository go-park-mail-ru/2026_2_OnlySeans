package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"maps"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
	"github.com/jackc/pgx/v5/pgconn"
)

// Operation records one repository operation, including transaction completion.
// Fields must contain identifiers and metadata, never SQL arguments or session tokens.
type Operation struct {
	Stage    string
	NotFound bool
	log      logger.Logger
	ctx      context.Context
	fields   map[string]any
	started  time.Time
}

func StartOperation(ctx context.Context, log logger.Logger, backend, operation string, fields map[string]any) *Operation {
	attrs := make(map[string]any, len(fields)+6)
	maps.Copy(attrs, fields)
	attrs["backend"] = backend
	attrs["operation"] = operation
	op := &Operation{Stage: "start", log: log, ctx: ctx, fields: attrs, started: time.Now()}
	if log != nil {
		log.DebugContext(ctx, "database operation started", maps.Clone(attrs))
	}
	return op
}

func (op *Operation) Finish(err error) {
	if op.log == nil {
		return
	}
	fields := maps.Clone(op.fields)
	fields["duration_ms"] = float64(time.Since(op.started).Microseconds()) / 1000
	fields["stage"] = op.Stage
	fields["outcome"] = "ok"
	if err == nil {
		if op.NotFound {
			fields["outcome"] = "not_found"
		}
		op.log.InfoContext(op.ctx, "database operation completed", fields)
		return
	}
	fields["outcome"] = "error"
	for _, missing := range []error{sql.ErrNoRows, domain.ErrUserNotFound, domain.ErrFilmNotFound, domain.ErrPersonNotFound, domain.ErrCollectionNotFound, domain.ErrReviewNotFound, domain.ErrFileNotFound, domain.ErrSessionNotFound, domain.ErrRatingNotFound} {
		if errors.Is(err, missing) {
			fields["outcome"] = "not_found"
			op.log.InfoContext(op.ctx, "database operation completed", fields)
			return
		}
	}
	for _, rejected := range []error{domain.ErrUserExists, domain.ErrUsernameTaken, domain.ErrReviewExists, domain.ErrVersionConflict, domain.ErrInvalidLibraryData, domain.ErrInvalidFilmRating} {
		if errors.Is(err, rejected) {
			fields["outcome"] = "rejected"
			fields["error"] = rejected.Error()
			op.log.WarnContext(op.ctx, "database operation rejected", fields)
			return
		}
	}
	// PostgreSQL diagnostics may include passwords or other SQL argument values.
	// Keep the SQLSTATE, never Message, Detail or the SQL text.
	var pgErr *pgconn.PgError
	switch {
	case errors.As(err, &pgErr):
		fields["sqlstate"] = pgErr.Code
		fields["error"] = "postgres error: SQLSTATE " + pgErr.Code
	case errors.Is(err, context.Canceled):
		fields["error"] = context.Canceled.Error()
	case errors.Is(err, context.DeadlineExceeded):
		fields["error"] = context.DeadlineExceeded.Error()
	default:
		fields["error"] = "database operation failed"
		fields["error_type"] = fmt.Sprintf("%T", errors.Unwrap(err))
		if errors.Unwrap(err) == nil {
			fields["error_type"] = fmt.Sprintf("%T", err)
		}
	}
	op.log.ErrorContext(op.ctx, "database operation failed", fields)
}
