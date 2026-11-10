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
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

const AvatarNameRandomBytes = imageNameRandomBytes

type UploadAvatarOptions struct {
	MaxSize        int64
	MaxDimension   int
	CleanupTimeout time.Duration
}

var (
	ErrAvatarMissing            = errors.New("avatar file is required")
	ErrInvalidAvatarSize        = errors.New("avatar size must be positive")
	ErrAvatarTooLarge           = errors.New("avatar file is too large")
	ErrInvalidAvatarImage       = errors.New("avatar must be a valid JPEG or PNG image")
	ErrAvatarDimensionsTooLarge = errors.New("avatar dimensions exceed the allowed limit")
)

type UploadAvatarInput struct {
	UserID domain.UserID
	Body   io.ReadSeeker
	Size   int64
}

type UploadAvatarOutput struct {
	Filename string
}

type AvatarStorage interface {
	Delete(ctx context.Context, key string) error
	Upload(
		ctx context.Context,
		key string,
		contentType string,
		body io.ReadSeeker,
	) error
}

type AvatarWriter interface {
	SaveAvatar(ctx context.Context, userID domain.UserID, file domain.AvatarFile) error
}

type UploadAvatarUseCase struct {
	storage AvatarStorage
	avatars AvatarWriter
	options UploadAvatarOptions
	logger  logger.Logger
}

func NewUploadAvatarUseCase(storage AvatarStorage, avatars AvatarWriter, options UploadAvatarOptions, appLogger logger.Logger) (*UploadAvatarUseCase, error) {
	if appLogger == nil {
		return nil, domain.ErrInvalidAppLogger
	}
	if storage == nil {
		return nil, errors.New("avatar storage is nil")
	}
	if avatars == nil {
		return nil, errors.New("avatar repository is nil")
	}
	if options.MaxSize <= 0 || options.MaxSize > math.MaxInt64-multipartOverheadBytes || options.MaxDimension <= 0 || options.CleanupTimeout <= 0 {
		return nil, errors.New("avatar limits and cleanup timeout must be positive and size must leave room for multipart overhead")
	}
	return &UploadAvatarUseCase{
		options: options,
		logger:  appLogger,
		storage: storage,
		avatars: avatars,
	}, nil
}

func (uc *UploadAvatarUseCase) MaxSize() int64 {
	return uc.options.MaxSize
}

func (uc *UploadAvatarUseCase) Run(ctx context.Context, input UploadAvatarInput) (*UploadAvatarOutput, error) {
	uc.logger.DebugContext(ctx, "avatar upload started", map[string]any{"user_id": input.UserID, "size": input.Size})
	if input.UserID <= 0 {
		uc.logger.WarnContext(ctx, "avatar upload rejected: invalid user", nil)
		return nil, domain.ErrUnauthorized
	}
	imageErrors := imageUploadErrors{
		missing:            ErrAvatarMissing,
		invalidSize:        ErrInvalidAvatarSize,
		tooLarge:           ErrAvatarTooLarge,
		invalidImage:       ErrInvalidAvatarImage,
		dimensionsTooLarge: ErrAvatarDimensionsTooLarge,
	}
	image, err := readUploadImage(input.Body, input.Size, uc.options.MaxSize, uc.options.MaxDimension, imageErrors)
	if err != nil {
		fields := map[string]any{"user_id": input.UserID, "error": err, "size": input.Size}
		if imageErrors.contains(err) {
			uc.logger.WarnContext(ctx, "avatar upload rejected: invalid file", fields)
		} else {
			uc.logger.ErrorContext(ctx, "read avatar failed", fields)
		}
		return nil, err
	}
	uc.logger.DebugContext(ctx, "avatar image validated", map[string]any{
		"user_id": input.UserID,
		"size":    len(image.data),
		"width":   image.width,
		"height":  image.height,
	})

	filename, err := newImageFilename(image.extension)
	if err != nil {
		uc.logger.ErrorContext(ctx, "generate avatar name failed", map[string]any{"user_id": input.UserID, "error": err})
		return nil, err
	}
	data, contentType := image.data, image.contentType

	if err := uc.storage.Upload(
		ctx,
		filename,
		contentType,
		bytes.NewReader(data),
	); err != nil {
		uc.logger.ErrorContext(ctx, "upload avatar object failed", map[string]any{"user_id": input.UserID, "filename": filename, "error": err})
		return nil, fmt.Errorf("save avatar: %w", err)
	}

	uc.logger.DebugContext(ctx, "avatar object uploaded", map[string]any{"user_id": input.UserID, "filename": filename, "size": len(data)})
	err = uc.avatars.SaveAvatar(ctx, input.UserID, domain.AvatarFile{
		Filename:    filename,
		ContentType: contentType,
		Size:        int64(len(data)),
	})
	if err != nil {
		uc.logger.ErrorContext(ctx, "save account avatar failed", map[string]any{"user_id": input.UserID, "filename": filename, "error": err})
		if !errors.Is(err, domain.ErrAvatarCommitUnknown) {
			cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), uc.options.CleanupTimeout)
			cleanupErr := uc.storage.Delete(cleanupCtx, filename)
			cancel()
			if cleanupErr != nil {
				uc.logger.ErrorContext(ctx, "remove unlinked avatar object failed", map[string]any{"user_id": input.UserID, "filename": filename, "error": cleanupErr})
				return nil, errors.Join(fmt.Errorf("attach avatar: %w", err), cleanupErr)
			}
			uc.logger.DebugContext(ctx, "unlinked avatar object removed", map[string]any{"user_id": input.UserID, "filename": filename})
		} else {
			uc.logger.WarnContext(ctx, "avatar object retained: commit outcome unknown", map[string]any{"user_id": input.UserID, "filename": filename})
		}
		return nil, fmt.Errorf("attach avatar: %w", err)
	}

	uc.logger.InfoContext(ctx, "account avatar saved", map[string]any{"user_id": input.UserID, "filename": filename, "size": len(data), "content_type": contentType})
	return &UploadAvatarOutput{
		Filename: filename,
	}, nil
}
