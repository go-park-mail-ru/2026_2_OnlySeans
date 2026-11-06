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
}

func NewUploadAvatarUseCase(storage AvatarStorage, avatars AvatarWriter, options UploadAvatarOptions) (*UploadAvatarUseCase, error) {
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
		storage: storage,
		avatars: avatars,
	}, nil
}

func (uc *UploadAvatarUseCase) MaxSize() int64 {
	return uc.options.MaxSize
}

func (uc *UploadAvatarUseCase) Run(ctx context.Context, input UploadAvatarInput) (*UploadAvatarOutput, error) {
	if input.UserID <= 0 {
		return nil, domain.ErrUnauthorized
	}
	if input.Body == nil {
		return nil, ErrAvatarMissing
	}
	if input.Size <= 0 {
		return nil, ErrInvalidAvatarSize
	}
	if input.Size > uc.options.MaxSize {
		return nil, ErrAvatarTooLarge
	}
	if _, err := input.Body.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("rewind avatar: %w", err)
	}

	data, err := io.ReadAll(io.LimitReader(input.Body, uc.options.MaxSize+1))
	if err != nil {
		return nil, fmt.Errorf("read avatar: %w", err)
	}

	if len(data) == 0 {
		return nil, ErrInvalidAvatarSize
	}

	if int64(len(data)) > uc.options.MaxSize {
		return nil, ErrAvatarTooLarge
	}

	imageConfig, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || (format != "jpeg" && format != "png") {
		return nil, ErrInvalidAvatarImage
	}

	if imageConfig.Width <= 0 || imageConfig.Height <= 0 {
		return nil, ErrInvalidAvatarImage
	}

	if imageConfig.Width > uc.options.MaxDimension ||
		imageConfig.Height > uc.options.MaxDimension {
		return nil, ErrAvatarDimensionsTooLarge
	}

	if _, _, err := image.Decode(bytes.NewReader(data)); err != nil {
		return nil, ErrInvalidAvatarImage
	}

	randomBytes := make([]byte, AvatarNameRandomBytes)
	if _, err := rand.Read(randomBytes); err != nil {
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
		return nil, fmt.Errorf("save avatar: %w", err)
	}

	err = uc.avatars.SaveAvatar(ctx, input.UserID, domain.AvatarFile{
		Filename:    filename,
		ContentType: contentType,
		Size:        int64(len(data)),
	})
	if err != nil {
		if !errors.Is(err, domain.ErrAvatarCommitUnknown) {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), uc.options.CleanupTimeout)
			cleanupErr := uc.storage.Delete(cleanupCtx, filename)
			cancel()
			if cleanupErr != nil {
				return nil, errors.Join(fmt.Errorf("attach avatar: %w", err), cleanupErr)
			}
		}
		return nil, fmt.Errorf("attach avatar: %w", err)
	}

	return &UploadAvatarOutput{
		Filename: filename,
	}, nil
}
