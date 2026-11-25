package usecase

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/repository"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
)

var (
	ErrInvalidPosterFilmID      = ErrInvalidFilmID
	ErrPosterMissing            = errors.New("poster file is required")
	ErrInvalidPosterSize        = errors.New("poster size must be positive")
	ErrPosterTooLarge           = errors.New("poster file is too large")
	ErrInvalidPosterImage       = errors.New("poster must be a valid JPEG or PNG image")
	ErrPosterDimensionsTooLarge = errors.New("poster dimensions exceed the allowed limit")
)

type PosterUseCase interface {
	UploadPoster(ctx context.Context, input UploadPosterInput) (string, error)
	GetPosterURL(ctx context.Context, filmID domain.FilmID) (string, error)
	DeletePoster(ctx context.Context, filmID domain.FilmID) error
}

type UploadPosterInput struct {
	UserID domain.UserID
	FilmID domain.FilmID
	Body   io.ReadSeeker
	Size   int64
}

type PosterStorage interface {
	Upload(ctx context.Context, key, contentType string, body io.ReadSeeker) error
	GetURL(ctx context.Context, key string) (string, error)
	Delete(ctx context.Context, key string) error
}

type PosterOptions struct {
	MaxSize        int64
	MaxDimension   int
	CleanupTimeout time.Duration
}

type posterUseCase struct {
	posters   repository.PosterRepository
	films     repository.FilmReader
	storage   PosterStorage
	options   PosterOptions
	appLogger logger.Logger
}

func NewPosterUseCase(posters repository.PosterRepository, films repository.FilmReader, storage PosterStorage, options PosterOptions, appLogger logger.Logger) (PosterUseCase, error) {
	if posters == nil || films == nil || storage == nil || appLogger == nil {
		return nil, errors.New("poster repository, film reader, storage and logger are required")
	}

	if options.MaxSize <= 0 || options.MaxSize > math.MaxInt64-multipartOverheadBytes || options.MaxDimension <= 0 || options.CleanupTimeout <= 0 {
		return nil, errors.New("poster limits and cleanup timeout must be positive and size must leave room for multipart overhead")
	}

	return &posterUseCase{
		posters:   posters,
		films:     films,
		storage:   storage,
		options:   options,
		appLogger: appLogger,
	}, nil
}

func (u *posterUseCase) UploadPoster(ctx context.Context, input UploadPosterInput) (string, error) {
	if input.UserID <= 0 {
		return "", domain.ErrUnauthorized
	}

	if input.FilmID <= 0 {
		return "", ErrInvalidPosterFilmID
	}

	if err := ctx.Err(); err != nil {
		return "", err
	}

	if _, err := u.films.GetFilmByID(ctx, input.FilmID); err != nil {
		return "", fmt.Errorf("get film for poster: %w", err)
	}

	image, err := readUploadImage(input.Body, input.Size, u.options.MaxSize, u.options.MaxDimension)
	if err != nil {
		return "", err
	}

	filename, err := newImageFilename(image.extension)
	if err != nil {
		return "", err
	}

	if err := u.storage.Upload(ctx, filename, image.contentType, bytes.NewReader(image.data)); err != nil {
		return "", fmt.Errorf("upload poster: %w", err)
	}

	err = u.posters.SavePoster(ctx, input.FilmID, input.UserID, domain.PosterFile{
		Filename:    filename,
		ContentType: image.contentType,
		Size:        int64(len(image.data)),
	})
	if err != nil {
		saveErr := fmt.Errorf("attach poster: %w", err)

		if errors.Is(err, repository.ErrCommitUnknown) {
			u.appLogger.WarnContext(ctx, "poster object retained: commit outcome unknown", map[string]any{
				"filename": filename,
				"error":    err,
			})

			return "", saveErr
		}

		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), u.options.CleanupTimeout)
		cleanupErr := u.storage.Delete(cleanupCtx, filename)

		cancel()

		if cleanupErr != nil {
			u.appLogger.ErrorContext(ctx, "remove unlinked poster object failed", map[string]any{
				"filename": filename,
				"error":    cleanupErr,
			})

			return "", errors.Join(saveErr, fmt.Errorf("remove unlinked poster: %w", cleanupErr))
		}

		return "", saveErr
	}

	return filename, nil
}

func (u *posterUseCase) GetPosterURL(ctx context.Context, filmID domain.FilmID) (string, error) {
	if filmID <= 0 {
		return "", ErrInvalidPosterFilmID
	}

	poster, err := u.posters.GetPoster(ctx, filmID)
	if err != nil {
		return "", fmt.Errorf("get film poster: %w", err)
	}

	if poster == nil {
		return "", domain.ErrFileNotFound
	}

	if poster.ExternalURL != nil {
		return *poster.ExternalURL, nil
	}

	url, err := u.storage.GetURL(ctx, poster.Filename)
	if err != nil {
		return "", fmt.Errorf("get poster URL: %w", err)
	}

	return url, nil
}

func (u *posterUseCase) DeletePoster(ctx context.Context, filmID domain.FilmID) error {
	if filmID <= 0 {
		return ErrInvalidPosterFilmID
	}

	if err := u.posters.DeletePoster(ctx, filmID); err != nil {
		return fmt.Errorf("delete film poster: %w", err)
	}

	return nil
}
