package files

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
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

type imageUploadErrors struct {
	missing            error
	invalidSize        error
	tooLarge           error
	invalidImage       error
	dimensionsTooLarge error
}

func (e imageUploadErrors) contains(err error) bool {
	return errors.Is(err, e.missing) ||
		errors.Is(err, e.invalidSize) ||
		errors.Is(err, e.tooLarge) ||
		errors.Is(err, e.invalidImage) ||
		errors.Is(err, e.dimensionsTooLarge)
}

type validatedImage struct {
	data        []byte
	contentType string
	extension   string
	width       int
	height      int
}

func readUploadImage(body io.ReadSeeker, declaredSize, maxSize int64, maxDimension int, failures imageUploadErrors) (*validatedImage, error) {
	if body == nil {
		return nil, failures.missing
	}
	if declaredSize <= 0 {
		return nil, failures.invalidSize
	}
	if declaredSize > maxSize {
		return nil, failures.tooLarge
	}
	if _, err := body.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("rewind image: %w", err)
	}

	data, err := io.ReadAll(io.LimitReader(body, maxSize+1))
	if err != nil {
		return nil, fmt.Errorf("read image: %w", err)
	}
	if len(data) == 0 {
		return nil, failures.invalidSize
	}
	if int64(len(data)) > maxSize {
		return nil, failures.tooLarge
	}

	imageConfig, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || (format != "jpeg" && format != "png") {
		return nil, failures.invalidImage
	}
	if imageConfig.Width <= 0 || imageConfig.Height <= 0 {
		return nil, failures.invalidImage
	}
	if imageConfig.Width > maxDimension || imageConfig.Height > maxDimension {
		return nil, failures.dimensionsTooLarge
	}
	if _, _, err := image.Decode(bytes.NewReader(data)); err != nil {
		return nil, failures.invalidImage
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
