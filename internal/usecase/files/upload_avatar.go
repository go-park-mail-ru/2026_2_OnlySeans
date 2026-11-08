package files

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"io"
	"math"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"

	_ "image/jpeg"
	_ "image/png"
)

const AvatarNameRandomBytes = 16

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
	if options.MaxSize <= 0 || options.MaxSize > math.MaxInt64-(1<<20) || options.MaxDimension <= 0 || options.CleanupTimeout <= 0 {
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
	uc.logger.Debug("avatar upload started", map[string]any{"user_id": input.UserID, "size": input.Size})
	if input.UserID <= 0 {
		uc.logger.Warn("avatar upload rejected: invalid user", nil)
		return nil, domain.ErrUnauthorized
	}
	if input.Body == nil {
		uc.logger.Warn("avatar upload rejected: missing file", map[string]any{"user_id": input.UserID})
		return nil, ErrAvatarMissing
	}
	if input.Size <= 0 {
		uc.logger.Warn("avatar upload rejected: invalid declared size", map[string]any{"user_id": input.UserID, "size": input.Size})
		return nil, ErrInvalidAvatarSize
	}
	if input.Size > uc.options.MaxSize {
		uc.logger.Warn("avatar upload rejected: declared size exceeds limit", map[string]any{"user_id": input.UserID, "size": input.Size, "max_size": uc.options.MaxSize})
		return nil, ErrAvatarTooLarge
	}
	if _, err := input.Body.Seek(0, io.SeekStart); err != nil {
		uc.logger.Error("rewind avatar failed", map[string]any{"user_id": input.UserID, "error": err})
		return nil, fmt.Errorf("rewind avatar: %w", err)
	}

	data, err := io.ReadAll(io.LimitReader(input.Body, uc.options.MaxSize+1))
	if err != nil {
		uc.logger.Error("read avatar failed", map[string]any{"user_id": input.UserID, "error": err})
		return nil, fmt.Errorf("read avatar: %w", err)
	}

	if len(data) == 0 {
		uc.logger.Warn("avatar upload rejected: empty file", map[string]any{"user_id": input.UserID})
		return nil, ErrInvalidAvatarSize
	}

	if int64(len(data)) > uc.options.MaxSize {
		uc.logger.Warn("avatar upload rejected: actual size exceeds limit", map[string]any{"user_id": input.UserID, "size": len(data), "max_size": uc.options.MaxSize})
		return nil, ErrAvatarTooLarge
	}

	imageConfig, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || (format != "jpeg" && format != "png") {
		uc.logger.Warn("avatar upload rejected: invalid image format", map[string]any{"user_id": input.UserID, "format": format, "error": err})
		return nil, ErrInvalidAvatarImage
	}

	if imageConfig.Width <= 0 || imageConfig.Height <= 0 {
		uc.logger.Warn("avatar upload rejected: invalid dimensions", map[string]any{"user_id": input.UserID, "width": imageConfig.Width, "height": imageConfig.Height})
		return nil, ErrInvalidAvatarImage
	}

	if imageConfig.Width > uc.options.MaxDimension ||
		imageConfig.Height > uc.options.MaxDimension {
		uc.logger.Warn("avatar upload rejected: dimensions exceed limit", map[string]any{"user_id": input.UserID, "width": imageConfig.Width, "height": imageConfig.Height, "max_dimension": uc.options.MaxDimension})
		return nil, ErrAvatarDimensionsTooLarge
	}

	if _, _, err := image.Decode(bytes.NewReader(data)); err != nil {
		uc.logger.Warn("avatar upload rejected: damaged image", map[string]any{"user_id": input.UserID, "error": err})
		return nil, ErrInvalidAvatarImage
	}

	randomBytes := make([]byte, AvatarNameRandomBytes)
	if _, err := rand.Read(randomBytes); err != nil {
		uc.logger.Error("generate avatar name failed", map[string]any{"user_id": input.UserID, "error": err})
		return nil, fmt.Errorf("generate avatar name: %w", err)
	}

	extension := ".jpg"
	contentType := "image/jpeg"

	if format == "png" {
		extension = ".png"
		contentType = "image/png"
	}

	filename := hex.EncodeToString(randomBytes) + extension

	if err := uc.storage.Upload(
		ctx,
		filename,
		contentType,
		bytes.NewReader(data),
	); err != nil {
		uc.logger.Error("upload avatar object failed", map[string]any{"user_id": input.UserID, "filename": filename, "error": err})
		return nil, fmt.Errorf("save avatar: %w", err)
	}

	uc.logger.Debug("avatar object uploaded", map[string]any{"user_id": input.UserID, "filename": filename, "size": len(data)})
	err = uc.avatars.SaveAvatar(ctx, input.UserID, domain.AvatarFile{
		Filename:    filename,
		ContentType: contentType,
		Size:        int64(len(data)),
	})
	if err != nil {
		uc.logger.Error("save account avatar failed", map[string]any{"user_id": input.UserID, "filename": filename, "error": err})
		if !errors.Is(err, domain.ErrAvatarCommitUnknown) {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), uc.options.CleanupTimeout)
			cleanupErr := uc.storage.Delete(cleanupCtx, filename)
			cancel()
			if cleanupErr != nil {
				uc.logger.Error("remove unlinked avatar object failed", map[string]any{"user_id": input.UserID, "filename": filename, "error": cleanupErr})
				return nil, errors.Join(fmt.Errorf("attach avatar: %w", err), cleanupErr)
			}
			uc.logger.Debug("unlinked avatar object removed", map[string]any{"user_id": input.UserID, "filename": filename})
		} else {
			uc.logger.Warn("avatar object retained: commit outcome unknown", map[string]any{"user_id": input.UserID, "filename": filename})
		}
		return nil, fmt.Errorf("attach avatar: %w", err)
	}

	uc.logger.Info("account avatar saved", map[string]any{"user_id": input.UserID, "filename": filename, "size": len(data), "content_type": contentType})
	return &UploadAvatarOutput{
		Filename: filename,
	}, nil
}
