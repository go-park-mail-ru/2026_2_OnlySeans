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
	decimalNumberBase      = 10
	jsonDelimiterBytes     = 2
	jsonSeparatorBytes     = 1
	jsonShortEscapeBytes   = 2
	jsonUnicodeEscapeBytes = 6
	jsonControlBoundary    = 0x20
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

	return normalizedValueSize(value)
}

func normalizedValueSize(value any) (int, error) {
	switch typed := value.(type) {
	case json.Number:
		number, err := normalizeJSONNumber(typed)
		if err != nil {
			return 0, err
		}

		return len(number), nil
	case string:
		return normalizedStringSize(typed), nil
	case nil:
		return len("null"), nil
	case bool:
		if typed {
			return len("true"), nil
		}

		return len("false"), nil
	case []any:
		size := jsonDelimiterBytes

		for index, item := range typed {
			normalized, err := normalizedValueSize(item)
			if err != nil {
				return 0, err
			}

			size += normalized
			if index > 0 {
				size += jsonSeparatorBytes
			}
		}

		return size, nil
	case map[string]any:
		size := jsonDelimiterBytes

		for key, item := range typed {
			normalized, err := normalizedValueSize(item)
			if err != nil {
				return 0, err
			}

			size += normalizedStringSize(key) + jsonSeparatorBytes + normalized + jsonSeparatorBytes
		}

		if len(typed) > 0 {
			size -= jsonSeparatorBytes
		}

		return size, nil
	}

	return 0, ErrInvalidJSON
}

func normalizedStringSize(value string) int {
	size := jsonDelimiterBytes

	for index := range len(value) {
		switch value[index] {
		case '"', '\\', '\b', '\f', '\n', '\r', '\t':
			size += jsonShortEscapeBytes
		default:
			if value[index] < jsonControlBoundary {
				size += jsonUnicodeEscapeBytes
			} else {
				size++
			}
		}
	}

	return size
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

	return shortestJSONNumber(sign, trimmed, &exponent), nil
}

func shortestJSONNumber(sign, digits string, exponent *big.Int) json.Number {
	scientific := sign + digits + "e" + exponent.String()

	var decimalSize big.Int
	if exponent.Sign() >= 0 {
		decimalSize.Add(exponent, big.NewInt(int64(len(sign)+len(digits))))
	} else {
		point := new(big.Int).Add(exponent, big.NewInt(int64(len(digits))))
		if point.Sign() > 0 {
			decimalSize.SetInt64(int64(len(sign) + len(digits) + len(".")))
		} else {
			decimalSize.Sub(big.NewInt(int64(len(sign)+len("0."))), exponent)
		}
	}

	if decimalSize.Cmp(big.NewInt(int64(len(scientific)))) > 0 {
		return json.Number(scientific)
	}

	if exponent.Sign() >= 0 {
		return json.Number(sign + digits + strings.Repeat("0", int(exponent.Int64())))
	}

	point := len(digits) + int(exponent.Int64())
	if point > 0 {
		return json.Number(sign + digits[:point] + "." + digits[point:])
	}

	return json.Number(sign + "0." + strings.Repeat("0", -point) + digits)
}
