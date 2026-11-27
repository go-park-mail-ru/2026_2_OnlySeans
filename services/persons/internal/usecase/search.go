package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/searchquery"
)

type PersonSearchRepository interface {
	SearchPersons(context.Context, SearchPersonsInput) (SearchPersonsOutput, error)
	CountPersons(context.Context, PersonSearchCriteria) (int64, error)
}

type PersonSearchUseCase struct {
	repo      PersonSearchRepository
	appLogger logger.Logger
}

func NewPersonSearchUseCase(repo PersonSearchRepository, appLogger logger.Logger) *PersonSearchUseCase {
	return &PersonSearchUseCase{repo: repo, appLogger: appLogger}
}

func (s *PersonSearchUseCase) SearchPersons(ctx context.Context, input SearchPersonsInput) (SearchPersonsOutput, error) {
	input.Criteria.Query = strings.TrimSpace(input.Criteria.Query)
	if err := input.Validate(); err != nil {
		return SearchPersonsOutput{}, fmt.Errorf("%w: %w", searchquery.ErrInvalidInput, err)
	}

	if input.Sort == "" {
		input.Sort = PersonSearchSortRelevance
	}

	result, err := s.repo.SearchPersons(ctx, input)
	if err != nil {
		return SearchPersonsOutput{}, fmt.Errorf("search persons: %w", err)
	}

	s.appLogger.DebugContext(ctx, "person search completed", map[string]any{
		"returned": len(result.Persons),
		"total":    result.Total,
	})

	return result, nil
}

func (s *PersonSearchUseCase) CountPersons(ctx context.Context, input CountPersonsInput) (CountPersonsOutput, error) {
	input.Criteria.Query = strings.TrimSpace(input.Criteria.Query)
	if err := input.Validate(); err != nil {
		return CountPersonsOutput{}, fmt.Errorf("%w: %w", searchquery.ErrInvalidInput, err)
	}

	total, err := s.repo.CountPersons(ctx, input.Criteria)
	if err != nil {
		return CountPersonsOutput{}, fmt.Errorf("count persons: %w", err)
	}

	s.appLogger.DebugContext(ctx, "person search count completed", map[string]any{"total": total})

	return CountPersonsOutput{Total: total}, nil
}
