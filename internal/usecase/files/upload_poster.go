package files

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/logging"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

var (
	ErrInvalidPosterFilmID      = errors.New("film id must be a positive integer")
	ErrPosterMissing            = errors.New("poster file is required")
	ErrInvalidPosterSize        = errors.New("poster size must be positive")
	ErrPosterTooLarge           = errors.New("poster file is too large")
	ErrInvalidPosterImage       = errors.New("poster must be a valid JPEG or PNG image")
	ErrPosterDimensionsTooLarge = errors.New("poster dimensions exceed the allowed limit")
)

type PosterStorage interface {
	Upload(ctx context.Context, key, contentType string, body io.ReadSeeker) error
	Delete(ctx context.Context, key string) error
}

type PosterFilmReader interface {
	GetFilmByID(ctx context.Context, id domain.FilmID) (domain.Film, error)
}

type UploadPosterOptions struct {
	MaxSize        int64
	MaxDimension   int
	CleanupTimeout time.Duration
}

type UploadPosterInput struct {
	UserID domain.UserID
	FilmID domain.FilmID
	Body   io.ReadSeeker
	Size   int64
}

type UploadPosterOutput struct {
	Filename string
}

type UploadPosterUseCase struct {
	storage PosterStorage
	posters domain.PosterRepository
	films   PosterFilmReader
	options UploadPosterOptions
	logger  logger.Logger
}

func NewUploadPosterUseCase(storage PosterStorage, posters domain.PosterRepository, films PosterFilmReader, options UploadPosterOptions, appLogger logger.Logger) (*UploadPosterUseCase, error) {
	if appLogger == nil {
		return nil, domain.ErrInvalidAppLogger
	}
	if storage == nil {
		return nil, errors.New("poster storage is nil")
	}
	if posters == nil {
		return nil, errors.New("poster repository is nil")
	}
	if films == nil {
		return nil, domain.ErrNilFilmsRepo
	}
	if options.MaxSize <= 0 || options.MaxSize > math.MaxInt64-multipartOverheadBytes || options.MaxDimension <= 0 || options.CleanupTimeout <= 0 {
		return nil, errors.New("poster limits and cleanup timeout must be positive and size must leave room for multipart overhead")
	}
	return &UploadPosterUseCase{
		storage: storage,
		posters: posters,
		films:   films,
		options: options,
		logger:  appLogger,
	}, nil
}

func (uc *UploadPosterUseCase) MaxSize() int64 {
	return uc.options.MaxSize
}

func (uc *UploadPosterUseCase) Run(ctx context.Context, input UploadPosterInput) (*UploadPosterOutput, error) {
	fields := map[string]any{"user_id": input.UserID, "film_id": input.FilmID}
	uc.logger.DebugContext(ctx, "poster upload started", fields)
	if input.UserID <= 0 {
		uc.logger.WarnContext(ctx, "poster upload rejected: invalid user", fields)
		return nil, domain.ErrUnauthorized
	}
	if input.FilmID <= 0 {
		uc.logger.WarnContext(ctx, "poster upload rejected: invalid film ID", fields)
		return nil, ErrInvalidPosterFilmID
	}
	if err := ctx.Err(); err != nil {
		uc.logger.DebugContext(ctx, "poster upload canceled", fields)
		return nil, err
	}

	if _, err := uc.films.GetFilmByID(ctx, input.FilmID); err != nil {
		logging.Failure(ctx, uc.logger, "get film for poster failed", err, fields)
		return nil, fmt.Errorf("get film for poster: %w", err)
	}
	uc.logger.DebugContext(ctx, "poster film found", fields)

	imageErrors := imageUploadErrors{
		missing:            ErrPosterMissing,
		invalidSize:        ErrInvalidPosterSize,
		tooLarge:           ErrPosterTooLarge,
		invalidImage:       ErrInvalidPosterImage,
		dimensionsTooLarge: ErrPosterDimensionsTooLarge,
	}
	image, err := readUploadImage(input.Body, input.Size, uc.options.MaxSize, uc.options.MaxDimension, imageErrors)
	if err != nil {
		fields["error"] = err
		if imageErrors.contains(err) {
			uc.logger.WarnContext(ctx, "poster upload rejected: invalid file", fields)
		} else {
			uc.logger.ErrorContext(ctx, "read poster failed", fields)
		}
		return nil, err
	}
	uc.logger.DebugContext(ctx, "poster image validated", map[string]any{
		"user_id": input.UserID,
		"film_id": input.FilmID,
		"size":    len(image.data),
		"width":   image.width,
		"height":  image.height,
	})

	filename, err := newImageFilename(image.extension)
	if err != nil {
		fields["error"] = err
		uc.logger.ErrorContext(ctx, "generate poster name failed", fields)
		return nil, err
	}
	fields["filename"] = filename
	fields["size"] = len(image.data)
	fields["content_type"] = image.contentType

	if err := uc.storage.Upload(ctx, filename, image.contentType, bytes.NewReader(image.data)); err != nil {
		fields["error"] = err
		uc.logger.ErrorContext(ctx, "upload poster object failed", fields)
		return nil, fmt.Errorf("upload poster object: %w", err)
	}
	uc.logger.InfoContext(ctx, "poster object uploaded", fields)

	err = uc.posters.SavePoster(ctx, input.FilmID, input.UserID, domain.PosterFile{
		Filename:    filename,
		ContentType: image.contentType,
		Size:        int64(len(image.data)),
	})
	if err != nil {
		logging.Failure(ctx, uc.logger, "save film poster failed", err, fields)
		if errors.Is(err, domain.ErrPosterCommitUnknown) {
			uc.logger.WarnContext(ctx, "poster object retained: commit outcome unknown", fields)
			return nil, fmt.Errorf("attach poster: %w", err)
		}

		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), uc.options.CleanupTimeout)
		cleanupErr := uc.storage.Delete(cleanupCtx, filename)
		cancel()
		if cleanupErr != nil {
			fields["error"] = cleanupErr
			uc.logger.ErrorContext(ctx, "remove unlinked poster object failed", fields)
			return nil, errors.Join(fmt.Errorf("attach poster: %w", err), fmt.Errorf("remove unlinked poster: %w", cleanupErr))
		}
		uc.logger.DebugContext(ctx, "unlinked poster object removed", fields)
		return nil, fmt.Errorf("attach poster: %w", err)
	}

	uc.logger.InfoContext(ctx, "film poster saved", fields)
	return &UploadPosterOutput{Filename: filename}, nil
}
