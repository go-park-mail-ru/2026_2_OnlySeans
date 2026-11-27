package usecase

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/repository"
)

const maxTopGenresLimit = 100

var visitorPublicIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{16,64}$`)

type AnalyticsUseCase interface {
	TopGenres(ctx context.Context, from, to time.Time, limit int) ([]domain.GenreOpenStat, error)
	GenreAge(ctx context.Context, genreSlug string, from, to time.Time) ([]domain.AgeGroupOpenStat, error)
	RecordFilmOpen(ctx context.Context, visitorPublicID string, accountID *domain.UserID, filmID domain.FilmID) error
}

type analyticsUseCase struct {
	repository repository.AnalyticsRepository
	now        func() time.Time
}

func NewAnalyticsUseCase(repo repository.AnalyticsRepository) AnalyticsUseCase {
	return &analyticsUseCase{repository: repo, now: time.Now}
}

func (u *analyticsUseCase) TopGenres(ctx context.Context, from, to time.Time, limit int) ([]domain.GenreOpenStat, error) {
	if from.IsZero() || to.IsZero() || !from.Before(to) {
		return nil, domain.ErrInvalidAnalyticsPeriod
	}
	if limit < 1 || limit > maxTopGenresLimit {
		return nil, domain.ErrInvalidAnalyticsLimit
	}

	stats, err := u.repository.TopGenresByFilmOpens(ctx, from, to, limit)
	if err != nil {
		return nil, fmt.Errorf("top genres by film opens: %w", err)
	}

	return stats, nil
}

func (u *analyticsUseCase) GenreAge(ctx context.Context, genreSlug string, from, to time.Time) ([]domain.AgeGroupOpenStat, error) {
	genreSlug = strings.TrimSpace(genreSlug)
	if genreSlug == "" {
		return nil, domain.ErrInvalidAnalyticsGenreSlug
	}
	if from.IsZero() || to.IsZero() || !from.Before(to) {
		return nil, domain.ErrInvalidAnalyticsPeriod
	}

	stats, err := u.repository.GenreOpensByAge(ctx, genreSlug, from, to)
	if err != nil {
		return nil, fmt.Errorf("genre opens by age: %w", err)
	}

	return stats, nil
}

func (u *analyticsUseCase) RecordFilmOpen(ctx context.Context, visitorPublicID string, accountID *domain.UserID, filmID domain.FilmID) error {
	if !visitorPublicIDPattern.MatchString(visitorPublicID) || filmID <= 0 || (accountID != nil && *accountID <= 0) {
		return domain.ErrInvalidAnalyticsEvent
	}

	event := domain.FilmOpenEvent{
		VisitorPublicID: visitorPublicID,
		AccountID:       accountID,
		FilmID:          filmID,
		OccurredAt:      u.now().UTC(),
	}

	if err := u.repository.RecordFilmOpen(ctx, event); err != nil {
		return fmt.Errorf("record film open: %w", err)
	}

	return nil
}
