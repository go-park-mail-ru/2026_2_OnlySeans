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

const MaxDimension = 2000

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
