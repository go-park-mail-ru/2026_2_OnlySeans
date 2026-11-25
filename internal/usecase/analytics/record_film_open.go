package usecase

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/logging"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

var visitorPublicIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{16,64}$`)

func IsValidVisitorPublicID(value string) bool {
	return visitorPublicIDPattern.MatchString(value)
}

type RecordFilmOpenInput struct {
	VisitorPublicID string
	AccountID       *domain.UserID
	FilmID          domain.FilmID
}

type RecordFilmOpenUseCase struct {
	repository domain.AnalyticsEventRepository
	appLogger  logger.Logger
}

func NewRecordFilmOpenUseCase(repository domain.AnalyticsEventRepository, appLogger logger.Logger) (*RecordFilmOpenUseCase, error) {
	if repository == nil {
		return nil, domain.ErrNilAnalyticsRepo
	}

	if appLogger == nil {
		return nil, domain.ErrInvalidAppLogger
	}

	return &RecordFilmOpenUseCase{
		repository: repository,
		appLogger:  appLogger,
	}, nil
}

func (uc *RecordFilmOpenUseCase) Run(ctx context.Context, input RecordFilmOpenInput) error {
	if !IsValidVisitorPublicID(input.VisitorPublicID) ||
		input.FilmID <= 0 ||
		(input.AccountID != nil && *input.AccountID <= 0) {
		return domain.ErrInvalidAnalyticsEvent
	}

	event := domain.FilmOpenEvent{
		VisitorPublicID: input.VisitorPublicID,
		AccountID:       input.AccountID,
		FilmID:          input.FilmID,
		OccurredAt:      time.Now().UTC(),
	}

	if err := uc.repository.RecordFilmOpen(ctx, event); err != nil {
		logging.Failure(ctx, uc.appLogger, "record film open failed", err, map[string]any{
			"film_id":    input.FilmID,
			"account_id": input.AccountID,
		})
		return fmt.Errorf("record film open: %w", err)
	}

	uc.appLogger.InfoContext(ctx, "film open recorded", map[string]any{
		"film_id":    input.FilmID,
		"account_id": input.AccountID,
	})

	return nil
}
