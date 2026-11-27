package storage

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
	"github.com/jackc/pgx/v5/pgconn"
)

const UniqueViolation = "23505"

type Operation struct {
	Stage     string
	NotFound  bool
	Rejected  bool
	appLogger logger.Logger
	ctx       context.Context
	fields    map[string]any
	started   time.Time
}

func StartOperation(ctx context.Context, appLogger logger.Logger, backend, operation string, fields map[string]any) *Operation {
	attrs := make(map[string]any, len(fields)+6)
	maps.Copy(attrs, fields)
	attrs["backend"] = backend
	attrs["operation"] = operation
	op := &Operation{Stage: "start", appLogger: appLogger, ctx: ctx, fields: attrs, started: time.Now()}
	if appLogger != nil {
		appLogger.DebugContext(ctx, "database operation started", maps.Clone(attrs))
	}
	return op
}

func (op *Operation) Finish(err error) {
	if op.appLogger == nil {
		return
	}

	fields := op.fields
	fields["duration_ms"] = float64(time.Since(op.started).Microseconds()) / 1000
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
