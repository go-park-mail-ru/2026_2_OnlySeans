package imageutil

import (
	"bytes"
	"errors"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

var (
	ErrInvalidImage  = errors.New("invalid image")
	ErrImageTooLarge = errors.New("image dimensions too large")
)

const (
	MaxDimension       = 2000
	MaxAvatarSize      = 5 << 20
	maxAvatarDimension = 4096
)

func Validate(data []byte) (string, error) {
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return "", ErrInvalidImage
	}

	if cfg.Width > MaxDimension || cfg.Height > MaxDimension {
		return "", ErrImageTooLarge
	}

	return format, nil
}

func ValidateAvatar(data []byte) (string, error) {
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || (format != "jpeg" && format != "png") || cfg.Width <= 0 || cfg.Height <= 0 {
		return "", ErrInvalidImage
	}
	if cfg.Width > maxAvatarDimension || cfg.Height > maxAvatarDimension {
		return "", ErrImageTooLarge
	}
	if _, _, err := image.Decode(bytes.NewReader(data)); err != nil {
		return "", ErrInvalidImage
	}

	return format, nil
}
