package httprequest

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

const MaxBodyBytes = 1 << 20

var ErrInvalidBody = errors.New("invalid request body")

func DecodeJSON(w http.ResponseWriter, r *http.Request, destination any) error {
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var raw json.RawMessage

	if err := decoder.Decode(&raw); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidBody, err)
	}

	if string(raw) == "null" {
		return ErrInvalidBody
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return ErrInvalidBody
	}

	valueDecoder := json.NewDecoder(bytes.NewReader(raw))
	valueDecoder.DisallowUnknownFields()

	if err := valueDecoder.Decode(destination); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidBody, err)
	}

	return nil
}
