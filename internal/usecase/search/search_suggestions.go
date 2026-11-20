package usecase

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/logging"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

type SearchSuggestionUseCase struct {
	repository domain.SearchRepository
	appLogger  logger.Logger
}

func NewSearchSuggestionUseCase(repository domain.SearchRepository, appLogger logger.Logger) (*SearchSuggestionUseCase, error) {
	if repository == nil {
		return nil, domain.ErrNilRepo
	}

	if appLogger == nil {
		return nil, domain.ErrInvalidAppLogger
	}

	return &SearchSuggestionUseCase{
		repository: repository,
		appLogger:  appLogger,
	}, nil
}

func (uc *SearchSuggestionUseCase) Run(ctx context.Context, input SearchSuggestionUsecaseInput) (*SearchSuggestionUsecaseOutput, error) {
	startedAt := time.Now()

	prepared, err := prepareSearchInput(input)
	if err != nil {
		logging.Validation(
			ctx, uc.appLogger,
			"search input validation failed",
			err,
			map[string]any{
				"query_length": utf8.RuneCountInString(
					strings.TrimSpace(input.Query),
				),
				"limit": input.Limit,
			},
		)

		return nil, err
	}

	fields := map[string]any{
		"query_length": utf8.RuneCountInString(prepared.Query),
		"limit":        prepared.Limit,
	}

	uc.appLogger.DebugContext(ctx, "search suggestions started", fields)

	films, err := uc.repository.SearchFilms(ctx, prepared.Query, prepared.Limit)
	if err != nil {
		logging.Failure(ctx, uc.appLogger, "search films failed", err, fields)
		return nil, err
	}

	persons, err := uc.repository.SearchPersons(ctx, prepared.Query, prepared.Limit)
	if err != nil {
		logging.Failure(ctx, uc.appLogger, "search persons failed", err, fields)
		return nil, err
	}

	uc.appLogger.DebugContext(ctx, "search suggestions completed", map[string]any{
		"query_length":  fields["query_length"],
		"limit":         prepared.Limit,
		"films_count":   len(films),
		"persons_count": len(persons),
		"duration_ms":   time.Since(startedAt).Milliseconds(),
	})

	return NewSearchSuggestionUsecaseOutput(films, persons), nil
}
