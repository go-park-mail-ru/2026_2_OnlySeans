package onlineconf_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	oc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain/onlineconf"
)

func TestConfigValidationJSONAndKnownLimits(t *testing.T) {
	for _, value := range []string{`{}`, `[]`, `"text"`, `123.5`, `true`, `null`} {
		config := oc.Config{Scope: "frontend", Key: "setting", Title: "Setting", Value: json.RawMessage(value)}
		if err := config.Validate(); err != nil {
			t.Fatal(value, err)
		}
	}

	for _, value := range []string{`null`, `49`, `20001`, `50.5`, `"500"`} {
		config := oc.Config{Scope: oc.ReviewsScope, Key: oc.ReviewMaxLengthKey, Title: "Limit", Value: json.RawMessage(value)}
		if err := config.Validate(); !errors.Is(err, oc.ErrInvalidReviewLimit) {
			t.Fatal(value, err)
		}
	}

	for _, value := range []string{`50`, `500`, `20000`} {
		config := oc.Config{Scope: oc.ReviewsScope, Key: oc.ReviewMaxLengthKey, Title: "Limit", Value: json.RawMessage(value)}
		if err := config.Validate(); err != nil {
			t.Fatal(value, err)
		}
	}

	config := oc.Config{Scope: "frontend", Key: "setting", Title: "Setting"}
	if err := config.Validate(); !errors.Is(err, oc.ErrInvalidJSON) {
		t.Fatal(err)
	}

	config.Value = json.RawMessage(`"` + strings.Repeat("x", oc.MaxValueBytes) + `"`)
	if err := config.Validate(); !errors.Is(err, oc.ErrValueTooLarge) {
		t.Fatal(err)
	}
}

func TestFlagValidation(t *testing.T) {
	flag := oc.FeatureFlag{Scope: "frontend", Key: "test", Title: "Test", Salt: "stable"}
	for _, percent := range []int{-1, 101} {
		flag.RolloutPercent = percent
		if err := flag.Validate(); !errors.Is(err, oc.ErrInvalidRollout) {
			t.Fatal(err)
		}
	}

	flag.RolloutPercent = 0
	flag.ForceOnUserIDs = []domain.UserID{1}
	flag.ForceOffUserIDs = []domain.UserID{1}

	if err := flag.Validate(); !errors.Is(err, oc.ErrConflictingOverrides) {
		t.Fatal(err)
	}
	flag.ForceOffUserIDs = nil
	flag.ForceOnUserIDs = []domain.UserID{0}

	if err := flag.Validate(); !errors.Is(err, oc.ErrInvalidUserID) {
		t.Fatal(err)
	}

	for _, key := range []string{"Frontend", "a.b", "a-b", "_key", "1key", strings.Repeat("a", 65)} {
		if err := oc.ValidateKey(key); !errors.Is(err, oc.ErrInvalidKey) {
			t.Fatal(key, err)
		}
	}
}
