package onlineconf

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

const (
	keyPatternExpression = `^[a-z][a-z0-9_]{0,63}$`
	MaxValueBytes        = 64 * 1024
)

var keyPattern = regexp.MustCompile(keyPatternExpression)

func ValidateKey(key string) error {
	if !keyPattern.MatchString(key) {
		return ErrInvalidKey
	}

	return nil
}

func validateIdentity(scope, key, title string) error {
	if err := ValidateKey(scope); err != nil {
		return fmt.Errorf("scope: %w", err)
	}

	if err := ValidateKey(key); err != nil {
		return err
	}

	if strings.TrimSpace(title) == "" {
		return ErrInvalidTitle
	}

	return nil
}

func (s Scope) Validate() error {
	if err := ValidateKey(s.Name); err != nil {
		return err
	}

	if strings.TrimSpace(s.Title) == "" {
		return ErrInvalidTitle
	}

	return nil
}

func (c Config) Validate() error {
	if err := validateIdentity(c.Scope, c.Key, c.Title); err != nil {
		return err
	}

	if err := ValidateConfigValue(c.Scope, c.Key, c.Value); err != nil {
		return err
	}

	return nil
}

func ValidateConfigValue(scope, key string, value json.RawMessage) error {
	if len(value) > MaxValueBytes {
		return ErrValueTooLarge
	}

	if !json.Valid(value) {
		return ErrInvalidJSON
	}

	if scope == ReviewsScope && key == ReviewMaxLengthKey {
		var limit *int
		if err := json.Unmarshal(value, &limit); err != nil || limit == nil || *limit < MinReviewLength || *limit > MaxReviewLength {
			return ErrInvalidReviewLimit
		}
	}

	return nil
}

func ValidateRollout(rollout int) error {
	if rollout < MinRolloutPercent || rollout > MaxRolloutPercent {
		return ErrInvalidRollout
	}

	return nil
}

func ValidateOverrides(overrides []Override) error {
	seen := make(map[domain.UserID]bool, len(overrides))

	for _, override := range overrides {
		if override.UserID < MinUserID || int64(override.UserID) > MaxSafeID {
			return ErrInvalidUserID
		}

		if enabled, exists := seen[override.UserID]; exists {
			if enabled != override.Enabled {
				return fmt.Errorf("user %d: %w", override.UserID, ErrConflictingOverrides)
			}

			return fmt.Errorf("user %d: %w", override.UserID, ErrDuplicateOverride)
		}

		seen[override.UserID] = override.Enabled
	}

	return nil
}

func (f FeatureFlag) Validate() error {
	if err := validateIdentity(f.Scope, f.Key, f.Title); err != nil {
		return err
	}

	if err := f.ValidateRules(); err != nil {
		return err
	}

	return nil
}

func (f FeatureFlag) ValidateRules() error {
	if err := ValidateRollout(f.RolloutPercent); err != nil {
		return err
	}

	if strings.TrimSpace(f.Salt) == "" {
		return ErrInvalidSalt
	}

	overrides := make([]Override, 0, len(f.ForceOnUserIDs)+len(f.ForceOffUserIDs))
	for _, id := range f.ForceOnUserIDs {
		overrides = append(overrides, Override{UserID: id, Enabled: true})
	}

	for _, id := range f.ForceOffUserIDs {
		overrides = append(overrides, Override{UserID: id, Enabled: false})
	}

	return ValidateOverrides(overrides)
}
