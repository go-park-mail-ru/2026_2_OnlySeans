package searchquery

import "fmt"

type FilterMode string

const (
	FilterAll FilterMode = "all"
	FilterAny FilterMode = "any"
)

// Filter excludes records that match any excluded value. An empty mode means all.
type Filter[T comparable] struct {
	Include []T
	Exclude []T
	Mode    FilterMode
}

func (f Filter[T]) Validate(maxValues int, valid func(T) bool) error {
	switch f.Mode {
	case "", FilterAll, FilterAny:
	default:
		return fmt.Errorf("mode must be all or any, got %q", f.Mode)
	}

	if err := ValidateValues(f.Include, maxValues, valid); err != nil {
		return fmt.Errorf("include: %w", err)
	}

	if err := ValidateValues(f.Exclude, maxValues, valid); err != nil {
		return fmt.Errorf("exclude: %w", err)
	}

	return nil
}

func ValidateValues[T comparable](values []T, maxValues int, valid func(T) bool) error {
	if len(values) > maxValues {
		return fmt.Errorf("must contain at most %d values", maxValues)
	}

	seen := make(map[T]struct{}, len(values))
	for _, value := range values {
		if !valid(value) {
			return fmt.Errorf("invalid value %v", value)
		}

		if _, exists := seen[value]; exists {
			return fmt.Errorf("duplicate value %v", value)
		}

		seen[value] = struct{}{}
	}

	return nil
}
