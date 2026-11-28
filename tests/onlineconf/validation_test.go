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

func TestHomeBlocksWriteSchema(t *testing.T) {
	banner := func(title string) string {
		return `[{"type":"banner","title":"` + title + `"}]`
	}
	button := func(href string) string {
		return `[{"type":"banner","title":"A","action":{"label":"Open","href":"` + href + `"}}]`
	}
	valid := []string{
		`[]`, `[{"type":"collection","id":1e2}]`, `[{"type":"collection","id":9007199254740991}]`,
		banner("  " + strings.Repeat("🎬", 50) + "  "), button("/"), button("/collections?sort=date#today"),
		button("/onlineconf-news"), button("/a/%2E%2e/collections"),
		"[" + strings.Repeat(`{"type":"collection","id":1},`, 5) + `{"type":"banner","title":"A","text":"B"}]`,
	}
	invalid := []string{
		`null`, `{}`, `[null]`, `[{}]`, `[{"type":"other"}]`, `[{"type":"banner"}]`, banner("   "), banner(strings.Repeat("🎬", 51)),
		`[{"type":"collection","id":0}]`, `[{"type":"collection","id":1.5}]`, `[{"type":"collection","id":"2"}]`,
		`[{"type":"collection","id":9007199254740992}]`, `[{"type":"banner","title":"A","text":null}]`,
		`[{"type":"banner","title":"A","text":"` + strings.Repeat("x", 301) + `"}]`,
		`[{"type":"banner","title":"A","action":null}]`, `[{"type":"banner","title":"A","action":{"href":"/"}}]`,
		`[{"type":"banner","title":"A","action":{"href":"/","label":"` + strings.Repeat("x", 41) + `"}}]`,
		button("collections"), button("//example.test"), button("/space here"), button("/onlineconf?from=home"),
		button("/old/%2e%2E/onlineconf/flags"), button("/a/..//other"), button(`/a\\b`),
		"[" + strings.Repeat(`{"type":"collection","id":1},`, 6) + `{"type":"banner","title":"A"}]`,
	}
	config := oc.Config{Scope: "frontend", Key: "home_blocks", Title: "Home"}
	for _, raw := range valid {
		config.Value = json.RawMessage(raw)
		if err := config.ValidateForWrite(); err != nil {
			t.Errorf("valid %s: %v", raw, err)
		}
	}
	for _, raw := range invalid {
		config.Value = json.RawMessage(raw)
		var blockError oc.HomeBlocksError
		if err := config.ValidateForWrite(); !errors.As(err, &blockError) {
			t.Errorf("invalid %s: got %v", raw, err)
		}
	}
	config.Scope, config.Value = "other", json.RawMessage(`null`)
	if err := config.ValidateForWrite(); err != nil {
		t.Fatal("unexpected schema outside frontend scope:", err)
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
