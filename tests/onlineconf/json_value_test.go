package onlineconf_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	oc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain/onlineconf"
	service "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/onlineconf"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/tests/testutil"
)

const (
	escapedRegressionASCIIBytes = 28000
	escapedRegressionAmpersands = 7000
)

func unescapedJSON(t *testing.T, value any) json.RawMessage {
	t.Helper()
	var output bytes.Buffer
	encoder := json.NewEncoder(&output)
	encoder.SetEscapeHTML(false)

	if err := encoder.Encode(value); err != nil {
		t.Fatal(err)
	}

	return bytes.TrimSpace(output.Bytes())
}

func TestConfigValueSizeStableAcrossEscapedRepresentations(t *testing.T) {
	for _, size := range []int{oc.MaxValueBytes - 1, oc.MaxValueBytes, oc.MaxValueBytes + 1} {
		text := strings.Repeat("&", size-len(`""`))
		raw := unescapedJSON(t, text)

		escaped, err := json.Marshal(text)
		if err != nil {
			t.Fatal(err)
		}

		for _, value := range []json.RawMessage{raw, escaped} {
			err := oc.ValidateConfigValue("frontend", "setting", value)
			if size <= oc.MaxValueBytes && err != nil {
				t.Fatalf("accepted boundary %d: %v", size, err)
			}

			if size > oc.MaxValueBytes && !errors.Is(err, oc.ErrValueTooLarge) {
				t.Fatalf("oversized boundary %d: %v", size, err)
			}
		}
	}
}

func TestConfigValueSizeStableAcrossJSONBNormalization(t *testing.T) {
	texts := []string{
		` { "message": "\u0026\u003c\u003e", "number": 1e65536 } `,
		`{"number":1` + strings.Repeat("0", 65536) + `,"message":"&<>"}`,
	}
	for _, text := range texts {
		if err := oc.ValidateConfigValue("frontend", "setting", json.RawMessage(text)); err != nil {
			t.Fatal("equivalent JSON rejected", err)
		}
	}

	for _, number := range []string{"0", "-0", "0e1000000", "1e+3", "1000.000", "0.00100", "-123.4500", "9007199254740993"} {
		if err := oc.ValidateConfigValue("frontend", "setting", json.RawMessage(number)); err != nil {
			t.Fatal(number, err)
		}
	}
}

func TestOnlineConfHTTPEscapedConfigPublishesValidSnapshot(t *testing.T) {
	repo, _ := newRepository(t)

	store, err := service.NewSnapshotStore(repo, service.SnapshotOptions{}, testutil.DiscardLogger())
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(func() { cancel(); <-store.Done() })

	if err := store.Start(ctx); err != nil {
		t.Fatal(err)
	}

	router := newHTTPRouterWithRefresher(t, repo, store, store)

	response := httpRequest(router, http.MethodPost, adminAPI+"/scopes", `{"name":"frontend","title":"Frontend"}`, "admin", "")
	if response.Code != http.StatusCreated {
		t.Fatal(response.Code, response.Body.String())
	}

	cases := []any{
		strings.Repeat("a", escapedRegressionASCIIBytes) + strings.Repeat("&", escapedRegressionAmpersands),
		strings.Repeat("&", oc.MaxValueBytes-len(`""`)),
		map[string]any{"<key>&": []any{"<>&", "\"\\\n\t", "\u2028\u2029"}, "large_integer": json.Number("9007199254740993")},
		json.Number("1e65536"),
	}
	for index, value := range cases {
		key := fmt.Sprintf("escaped_%d", index)
		body := unescapedJSON(t, map[string]any{"scope": "frontend", "key": key, "title": "Escaped value", "value": value})

		response = httpRequest(router, http.MethodPost, adminAPI+"/configs", string(body), "admin", "")
		if response.Code != http.StatusCreated {
			t.Fatalf("create %d: %d %s", index, response.Code, response.Body.String())
		}

		revision := int64(index + 2)

		waitForSnapshot(t, func() bool { snapshot, err := store.Current(); return err == nil && snapshot.Revision >= revision })

		response = httpRequest(router, http.MethodGet, oc.SnapshotAPIPath, "", "", accessTestToken)
		if response.Code != http.StatusOK {
			t.Fatal(response.Code, response.Body.String())
		}

		decoder := json.NewDecoder(response.Body)
		decoder.DisallowUnknownFields()

		var payload oc.SnapshotPayload

		if err := decoder.Decode(&payload); err != nil {
			t.Fatal(err)
		}

		if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
			t.Fatal(err)
		}

		if err := payload.Validate(); err != nil {
			t.Fatalf("201 published invalid snapshot: %v", err)
		}

		found := false

		for _, config := range payload.Configs {
			if config.Scope != "frontend" || config.Key != key {
				continue
			}

			found = true

			if index == 0 {
				if len(config.Value) <= oc.MaxValueBytes {
					t.Fatal("escaping regression was not reproduced")
				}

				var decoded string

				if err := json.Unmarshal(config.Value, &decoded); err != nil {
					t.Fatal(err)
				}

				if decoded != value {
					t.Fatal("string changed")
				}
			}

			if index == 2 {
				var decoded struct {
					LargeInteger json.Number `json:"large_integer"`
				}

				if err := json.Unmarshal(config.Value, &decoded); err != nil {
					t.Fatal(err)
				}

				if decoded.LargeInteger.String() != "9007199254740993" {
					t.Fatal("integer precision lost")
				}
			}
		}

		if !found {
			t.Fatal("created config missing from snapshot")
		}
	}

	oversized := unescapedJSON(t, map[string]any{"scope": "frontend", "key": "oversized", "title": "Oversized", "value": strings.Repeat("&", oc.MaxValueBytes-len(`""`)+1)})

	response = httpRequest(router, http.MethodPost, adminAPI+"/configs", string(oversized), "admin", "")
	if response.Code != http.StatusBadRequest {
		t.Fatal(response.Code, response.Body.String())
	}

	snapshot, err := repo.LoadSnapshot(ctx)
	if err != nil || snapshot.Revision != int64(len(cases)+1) {
		t.Fatal("oversized write changed revision", err)
	}
}
