package storage

import (
	"context"
	"database/sql"
	"errors"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
	sharedstorage "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/storage"
)

// The remaining monolith repositories use the shared recorder with their domain errors.
type ExternalObserver = sharedstorage.ExternalObserver

type OperationRecorder struct {
	recorder sharedstorage.OperationRecorder
}

func NewOperationRecorder(appLogger logger.Logger, observers ...ExternalObserver) OperationRecorder {
	return OperationRecorder{recorder: sharedstorage.NewOperationRecorder(appLogger, observers...)}
}

type Operation struct {
	*sharedstorage.Operation
}

func StartOperation(ctx context.Context, appLogger logger.Logger, backend, operation string, fields map[string]any) *Operation {
	return NewOperationRecorder(appLogger).Start(ctx, backend, operation, fields)
}

func (r OperationRecorder) Start(ctx context.Context, backend, operation string, fields map[string]any) *Operation {
	return &Operation{Operation: r.recorder.Start(ctx, backend, operation, fields)}
}

func (op *Operation) Finish(err error) {
	for _, missing := range []error{sql.ErrNoRows, domain.ErrUserNotFound, domain.ErrFilmNotFound, domain.ErrCollectionNotFound, domain.ErrReviewNotFound, domain.ErrFileNotFound, domain.ErrSessionNotFound, domain.ErrRatingNotFound} {
		if errors.Is(err, missing) {
			op.NotFound = true
			op.Operation.Finish(err)
			return
		}
	}
	for _, rejected := range []error{domain.ErrUserExists, domain.ErrUsernameTaken, domain.ErrReviewExists, domain.ErrVersionConflict, domain.ErrInvalidLibraryData, domain.ErrInvalidFilmRating} {
		if errors.Is(err, rejected) {
			op.Rejected = true
			op.Operation.Finish(rejected)
			return
		}
	}
	op.Operation.Finish(err)
}
