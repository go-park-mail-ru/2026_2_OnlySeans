package films

import (
	"context"
	"errors"
	"fmt"
	"regexp"
)

const (
	DefaultLimit = 20
	MaxLimit     = 100
)

var (
	ErrInvalidLimit  = fmt.Errorf("limit должен быть числом от 1 до %d", MaxLimit)
	ErrInvalidOffset = errors.New("offset должен быть неотрицательным числом")
	ErrInvalidSlug   = errors.New("некорректный slug подборки")
)

var slugPattern = regexp.MustCompile(`^[a-z0-9-]{3,128}$`)

type UseCase struct {
	repo Repository
}

func NewUseCase(repo Repository) *UseCase {
	return &UseCase{repo: repo}
}

func (uc *UseCase) ListFilms(ctx context.Context, limit, offset int) (FilmsPage, error) {
	if limit < 1 || limit > MaxLimit {
		return FilmsPage{}, ErrInvalidLimit
	}
	if offset < 0 {
		return FilmsPage{}, ErrInvalidOffset
	}

	films, total, err := uc.repo.ListFilms(ctx, limit, offset)
	if err != nil {
		return FilmsPage{}, fmt.Errorf("list films: %w", err)
	}

	return FilmsPage{Films: films, Total: total, Limit: limit, Offset: offset}, nil
}

func (uc *UseCase) ListCollections(ctx context.Context) ([]CollectionWithFilms, error) {
	collections, err := uc.repo.ListCollections(ctx)
	if err != nil {
		return nil, fmt.Errorf("list collections: %w", err)
	}

	result := make([]CollectionWithFilms, 0, len(collections))
	for _, collection := range collections {
		withFilms, err := uc.withFilms(ctx, collection)
		if err != nil {
			return nil, err
		}
		result = append(result, withFilms)
	}
	return result, nil
}

func (uc *UseCase) GetCollection(ctx context.Context, slug string) (CollectionWithFilms, error) {
	if !slugPattern.MatchString(slug) {
		return CollectionWithFilms{}, ErrInvalidSlug
	}

	collection, err := uc.repo.GetCollectionBySlug(ctx, slug)
	if err != nil {
		return CollectionWithFilms{}, fmt.Errorf("get collection %q: %w", slug, err)
	}
	return uc.withFilms(ctx, collection)
}

func (uc *UseCase) withFilms(ctx context.Context, collection Collection) (CollectionWithFilms, error) {
	films, err := uc.repo.ListCollectionFilms(ctx, collection.ID)
	if err != nil {
		return CollectionWithFilms{}, fmt.Errorf("list films of collection %q: %w", collection.Slug, err)
	}
	return CollectionWithFilms{Collection: collection, Films: films}, nil
}
