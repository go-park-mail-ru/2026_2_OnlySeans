package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"maps"
	"net"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
	"github.com/jackc/pgx/v5/pgconn"
)

type ExternalObserver interface {
	ObserveExternal(system, operation, outcome string, duration time.Duration)
}

type OperationRecorder struct {
	appLogger logger.Logger
	observer  ExternalObserver
}

func NewOperationRecorder(appLogger logger.Logger, observers ...ExternalObserver) OperationRecorder {
	recorder := OperationRecorder{appLogger: appLogger}
	if len(observers) > 0 {
		recorder.observer = observers[0]
	}
	return recorder
}

type Operation struct {
	Stage     string
	NotFound  bool
	appLogger logger.Logger
	observer  ExternalObserver
	backend   string
	name      string
	ctx       context.Context
	fields    map[string]any
	started   time.Time
}

func StartOperation(ctx context.Context, appLogger logger.Logger, backend, operation string, fields map[string]any) *Operation {
	return NewOperationRecorder(appLogger).Start(ctx, backend, operation, fields)
}

func (r OperationRecorder) Start(ctx context.Context, backend, operation string, fields map[string]any) *Operation {
	attrs := make(map[string]any, len(fields)+6)
	maps.Copy(attrs, fields)
	attrs["backend"] = backend
	attrs["operation"] = operation
	op := &Operation{
		Stage:     "start",
		appLogger: r.appLogger,
		observer:  r.observer,
		backend:   backend,
		name:      operation,
		ctx:       ctx,
		fields:    attrs,
		started:   time.Now(),
	}
	if r.appLogger != nil {
		r.appLogger.DebugContext(ctx, "database operation started", maps.Clone(attrs))
	}
	return op
}

func (op *Operation) Finish(err error) {
	if op.appLogger == nil && op.observer == nil {
		return
	}
	duration := time.Since(op.started)
	outcome, rejected := op.outcome(err)
	if op.observer != nil {
		op.observer.ObserveExternal(op.backend, op.name, outcome, duration)
	}
	if op.appLogger == nil {
		return
	}
	fields := maps.Clone(op.fields)
	fields["duration_ms"] = float64(duration.Microseconds()) / 1000
	fields["stage"] = op.Stage
	fields["outcome"] = outcome
	switch outcome {
	case "ok", "not_found":
		op.appLogger.InfoContext(op.ctx, "database operation completed", fields)
		return
	case "rejected":
		fields["error"] = rejected.Error()
		op.appLogger.WarnContext(op.ctx, "database operation rejected", fields)
		return
	}

	fields["outcome"] = "error"
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
	op.appLogger.ErrorContext(op.ctx, "database operation failed", fields)
}

func (op *Operation) outcome(err error) (string, error) {
	if err == nil {
		if op.NotFound {
			return "not_found", nil
		}
		return "ok", nil
	}
	for _, missing := range []error{sql.ErrNoRows, domain.ErrUserNotFound, domain.ErrFilmNotFound, domain.ErrPersonNotFound, domain.ErrCollectionNotFound, domain.ErrReviewNotFound, domain.ErrFileNotFound, domain.ErrSessionNotFound, domain.ErrRatingNotFound} {
		if errors.Is(err, missing) {
			return "not_found", nil
		}
	}
	for _, rejected := range []error{domain.ErrUserExists, domain.ErrUsernameTaken, domain.ErrReviewExists, domain.ErrVersionConflict, domain.ErrInvalidLibraryData, domain.ErrInvalidFilmRating} {
		if errors.Is(err, rejected) {
			return "rejected", rejected
		}
	}
	if errors.Is(err, context.Canceled) {
		return "canceled", nil
	}
	var timeout net.Error
	if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &timeout) && timeout.Timeout()) {
		return "timeout", nil
	}
	return "error", nil
}
