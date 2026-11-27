package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"maps"
	"net"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
	"github.com/jackc/pgx/v5/pgconn"
)

const UniqueViolation = "23505"

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
	Stage         string
	NotFound      bool
	Rejected      bool
	appLogger     logger.Logger
	ctx           context.Context
	fields        map[string]any
	started       time.Time
	observer      ExternalObserver
	backend, name string
}

func StartOperation(ctx context.Context, appLogger logger.Logger, backend, operation string, fields map[string]any) *Operation {
	return NewOperationRecorder(appLogger).Start(ctx, backend, operation, fields)
}

func (r OperationRecorder) Start(ctx context.Context, backend, operation string, fields map[string]any) *Operation {
	appLogger := r.appLogger
	attrs := make(map[string]any, len(fields)+6)
	maps.Copy(attrs, fields)
	attrs["backend"] = backend
	attrs["operation"] = operation
	op := &Operation{Stage: "start", appLogger: appLogger, ctx: ctx, fields: attrs, started: time.Now(), observer: r.observer, backend: backend, name: operation}
	if appLogger != nil {
		appLogger.DebugContext(ctx, "database operation started", maps.Clone(attrs))
	}
	return op
}

func (op *Operation) Finish(err error) {
	duration := time.Since(op.started)
	if op.observer != nil {
		op.observer.ObserveExternal(op.backend, op.name, op.outcome(err), duration)
	}
	if op.appLogger == nil {
		return
	}

	fields := maps.Clone(op.fields)
	fields["duration_ms"] = float64(duration.Microseconds()) / 1000
	fields["stage"] = op.Stage
	switch {
	case op.NotFound:
		fields["outcome"] = "not_found"
		op.appLogger.InfoContext(op.ctx, "database operation completed", fields)
	case err == nil:
		fields["outcome"] = "ok"
		op.appLogger.InfoContext(op.ctx, "database operation completed", fields)
	case op.Rejected:
		fields["outcome"] = "rejected"
		fields["error"] = err.Error()
		op.appLogger.WarnContext(op.ctx, "database operation rejected", fields)
	default:
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
			cause := errors.Unwrap(err)
			if cause == nil {
				cause = err
			}
			fields["error_type"] = fmt.Sprintf("%T", cause)
		}
		op.appLogger.ErrorContext(op.ctx, "database operation failed", fields)
	}
}

func (op *Operation) outcome(err error) string {
	var timeout net.Error
	switch {
	case op.NotFound || errors.Is(err, sql.ErrNoRows):
		return "not_found"
	case err == nil:
		return "ok"
	case op.Rejected:
		return "rejected"
	case errors.Is(err, context.Canceled):
		return "canceled"
	case errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &timeout) && timeout.Timeout()):
		return "timeout"
	default:
		return "error"
	}
}
