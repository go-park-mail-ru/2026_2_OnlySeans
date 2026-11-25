package usecase

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"image"
	"io"

	_ "image/jpeg"
	_ "image/png"
)

const (
	imageNameRandomBytes   = 16
	multipartOverheadBytes = 1 << 20
)

type validatedImage struct {
	data        []byte
	contentType string
	extension   string
	width       int
	height      int
}

func readUploadImage(body io.ReadSeeker, declaredSize, maxSize int64, maxDimension int) (*validatedImage, error) {
	if body == nil {
		return nil, ErrPosterMissing
	}

	if declaredSize <= 0 {
		return nil, ErrInvalidPosterSize
	}

	if declaredSize > maxSize {
		return nil, ErrPosterTooLarge
	}

	if _, err := body.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("rewind image: %w", err)
	}

	data, err := io.ReadAll(io.LimitReader(body, maxSize+1))
	if err != nil {
		return nil, fmt.Errorf("read image: %w", err)
	}

	if len(data) == 0 {
		return nil, ErrInvalidPosterSize
	}

	if int64(len(data)) > maxSize {
		return nil, ErrPosterTooLarge
	}

	imageConfig, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || (format != "jpeg" && format != "png") {
		return nil, ErrInvalidPosterImage
	}

	if imageConfig.Width <= 0 || imageConfig.Height <= 0 {
		return nil, ErrInvalidPosterImage
	}

	if imageConfig.Width > maxDimension || imageConfig.Height > maxDimension {
		return nil, ErrPosterDimensionsTooLarge
	}

	if _, _, err := image.Decode(bytes.NewReader(data)); err != nil {
		return nil, ErrInvalidPosterImage
	}

	result := &validatedImage{
		data:        data,
		contentType: "image/jpeg",
		extension:   ".jpg",
		width:       imageConfig.Width,
		height:      imageConfig.Height,
	}
	if format == "png" {
		result.contentType = "image/png"
		result.extension = ".png"
	}

	return result, nil
}

func newImageFilename(extension string) (string, error) {
	randomBytes := make([]byte, imageNameRandomBytes)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("generate image name: %w", err)
	}

	return hex.EncodeToString(randomBytes) + extension, nil
}
