package onlineconf

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"strings"
)

const (
	jsonValueTerminator = "\n"
	decimalNumberBase   = 10
)

func normalizedJSONSize(raw json.RawMessage) (int, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()

	var value any
	if err := decoder.Decode(&value); err != nil {
		return 0, fmt.Errorf("%w: %v", ErrInvalidJSON, err)
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return 0, ErrInvalidJSON
	}

	value, err := normalizeJSONNumbers(value)
	if err != nil {
		return 0, err
	}

	var normalized bytes.Buffer
	encoder := json.NewEncoder(&normalized)
	encoder.SetEscapeHTML(false)

	if err := encoder.Encode(value); err != nil {
		return 0, fmt.Errorf("normalize JSON value: %w", err)
	}

	return normalized.Len() - len(jsonValueTerminator), nil
}

func normalizeJSONNumbers(value any) (any, error) {
	switch typed := value.(type) {
	case json.Number:
		return normalizeJSONNumber(typed)
	case []any:
		for index, item := range typed {
			normalized, err := normalizeJSONNumbers(item)
			if err != nil {
				return nil, err
			}

			typed[index] = normalized
		}
	case map[string]any:
		for key, item := range typed {
			normalized, err := normalizeJSONNumbers(item)
			if err != nil {
				return nil, err
			}

			typed[key] = normalized
		}
	}

	return value, nil
}

func normalizeJSONNumber(number json.Number) (json.Number, error) {
	mantissa, exponentText, _ := strings.Cut(strings.ToLower(number.String()), "e")
	sign := ""

	if strings.HasPrefix(mantissa, "-") {
		sign = "-"
		mantissa = strings.TrimPrefix(mantissa, "-")
	}

	whole, fraction, _ := strings.Cut(mantissa, ".")

	digits := strings.TrimLeft(whole+fraction, "0")
	if digits == "" {
		return json.Number("0"), nil
	}

	trimmed := strings.TrimRight(digits, "0")

	exponentDigits := strings.TrimLeft(strings.TrimLeft(exponentText, "+-"), "0")
	if len(exponentDigits) >= MaxValueBytes {
		return "", ErrValueTooLarge
	}

	var exponent big.Int
	if exponentText != "" {
		if _, ok := exponent.SetString(exponentText, decimalNumberBase); !ok {
			return "", ErrInvalidJSON
		}
	}

	adjustment := int64(len(digits) - len(trimmed) - len(fraction))
	exponent.Add(&exponent, big.NewInt(adjustment))

	return json.Number(sign + trimmed + "e" + exponent.String()), nil
}
