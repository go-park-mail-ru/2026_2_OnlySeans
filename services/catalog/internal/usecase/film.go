package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/repository"
)

type FilmUseCase interface {
	CreateFilm(ctx context.Context, input domain.CreateFilmInput) (domain.FilmID, error)
	GetFilmByID(ctx context.Context, id domain.FilmID) (domain.Film, error)
	ListFilms(ctx context.Context, limit, offset int) ([]domain.Film, int, error)
	FilterFilms(ctx context.Context, filter domain.FilmFilter, limit, offset int) ([]domain.Film, int, error)
	ListGenres(ctx context.Context) ([]domain.Genre, error)
	UpdateFilm(ctx context.Context, id domain.FilmID, input domain.CreateFilmInput) error
	DeleteFilm(ctx context.Context, id domain.FilmID) error
}

type filmUseCase struct {
	reader repository.FilmReader
	lister repository.FilmLister
	genres repository.GenreReader
	writer repository.FilmWriter
}

func NewFilmUseCase(reader repository.FilmReader, lister repository.FilmLister, writer repository.FilmWriter) FilmUseCase {
	genres, _ := writer.(repository.GenreReader)
	return &filmUseCase{
		reader: reader,
		lister: lister,
		genres: genres,
		writer: writer,
	}
}

func (u *filmUseCase) ListFilms(ctx context.Context, limit, offset int) ([]domain.Film, int, error) {
	return u.FilterFilms(ctx, domain.FilmFilter{}, limit, offset)
}

func (u *filmUseCase) CreateFilm(ctx context.Context, input domain.CreateFilmInput) (domain.FilmID, error) {
	input.Title = strings.TrimSpace(input.Title)

	if err := input.Validate(); err != nil {
		return 0, err
	}

	id, err := u.writer.CreateFilm(ctx, input)
	if err != nil {
		return 0, fmt.Errorf("create film: %w", err)
	}

	return id, nil
}

func (u *filmUseCase) GetFilmByID(ctx context.Context, id domain.FilmID) (domain.Film, error) {
	if id <= 0 {
		return domain.Film{}, ErrInvalidFilmID
	}

	film, err := u.reader.GetFilmByID(ctx, id)
	if err != nil {
		return domain.Film{}, fmt.Errorf("get film: %w", err)
	}

	return film, nil
}

func (u *filmUseCase) FilterFilms(ctx context.Context, filter domain.FilmFilter, limit, offset int) ([]domain.Film, int, error) {
	if err := validatePagination(limit, offset); err != nil {
		return nil, 0, err
	}

	films, total, err := u.lister.FilterFilms(ctx, filter, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list films: %w", err)
	}

	return films, total, nil
}

func (u *filmUseCase) ListGenres(ctx context.Context) ([]domain.Genre, error) {
	if u.genres == nil {
		return nil, errors.New("genre reader is required")
	}

	genres, err := u.genres.ListGenres(ctx)
	if err != nil {
		return nil, fmt.Errorf("list genres: %w", err)
	}

	return genres, nil
}

func (u *filmUseCase) UpdateFilm(ctx context.Context, id domain.FilmID, input domain.CreateFilmInput) error {
	if id <= 0 {
		return ErrInvalidFilmID
	}

	input.Title = strings.TrimSpace(input.Title)

	if err := input.Validate(); err != nil {
		return err
	}

	if err := u.writer.UpdateFilm(ctx, id, input); err != nil {
		return fmt.Errorf("update film: %w", err)
	}

	return nil
}

func (u *filmUseCase) DeleteFilm(ctx context.Context, id domain.FilmID) error {
	if id <= 0 {
		return ErrInvalidFilmID
	}

	if err := u.writer.DeleteFilm(ctx, id); err != nil {
		return fmt.Errorf("delete film: %w", err)
	}

	return nil
}
