package onlineconf_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	delivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/onlineconf"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	oc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain/onlineconf"
	evaluator "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/onlineconfevaluator"
)

type contractTransport struct{ handler http.Handler }

func (t contractTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	recorder := httptest.NewRecorder()
	t.handler.ServeHTTP(recorder, request)

	return recorder.Result(), nil
}

func loadContractFixture(t *testing.T, name string) oc.SnapshotPayload {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	var payload oc.SnapshotPayload
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&payload); err != nil {
		t.Fatal(err)
	}

	if err := payload.Validate(); err != nil {
		t.Fatal(err)
	}

	return payload
}

func snapshotFromPayload(payload oc.SnapshotPayload) *oc.Snapshot {
	snapshot := &oc.Snapshot{Revision: payload.Revision, GeneratedAt: payload.GeneratedAt, Configs: make([]oc.Config, 0, len(payload.Configs)), Flags: make([]oc.FeatureFlag, 0, len(payload.Flags))}
	for _, config := range payload.Configs {
		snapshot.Configs = append(snapshot.Configs, oc.Config{Scope: config.Scope, Key: config.Key, Title: config.Key, Value: config.Value})
	}

	for _, flag := range payload.Flags {
		value := flag.AsFeatureFlag()
		value.Title = flag.Key
		snapshot.Flags = append(snapshot.Flags, value)
	}

	return snapshot
}

func TestSnapshotHTTPMatchesSharedContract(t *testing.T) {
	for _, fixture := range []string{"snapshot_empty.json", "snapshot_full.json"} {
		t.Run(fixture, func(t *testing.T) {
			expected := loadContractFixture(t, fixture)
			router := newHTTPRouter(t, &handlerRepository{}, &snapshotStub{value: snapshotFromPayload(expected)})
			client := http.Client{Transport: contractTransport{handler: router}}

			request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, accessTestOrigin+oc.SnapshotAPIPath, nil)
			if err != nil {
				t.Fatal(err)
			}

			request.Header.Set("Authorization", "Bearer "+accessTestToken)

			response, err := client.Do(request)
			if err != nil {
				t.Fatal(err)
			}

			t.Cleanup(func() {
				if err := response.Body.Close(); err != nil {
					t.Error(err)
				}
			})

			if response.StatusCode != http.StatusOK {
				t.Fatal(response.StatusCode)
			}

			decoder := json.NewDecoder(response.Body)
			decoder.DisallowUnknownFields()

			var received oc.SnapshotPayload

			if err := decoder.Decode(&received); err != nil {
				t.Fatal(err)
			}

			if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
				t.Fatal("trailing JSON", err)
			}

			if err := received.Validate(); err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(expected, received) {
				t.Fatalf("contract mismatch: %+v", received)
			}
		})
	}
}

func TestSharedEvaluatorMatchesAdminEndpoint(t *testing.T) {
	payload := loadContractFixture(t, "snapshot_full.json")
	source := &snapshotStub{}
	router := newHTTPRouter(t, &handlerRepository{}, source)

	for _, enabled := range []bool{false, true} {
		for _, percent := range []int{0, 25, 100} {
			payload.Flags[0].Enabled = enabled
			payload.Flags[0].RolloutPercent = percent
			source.value = snapshotFromPayload(payload)

			for _, userID := range []domain.UserID{1, 2, 101, 201, 202} {
				flag := payload.Flags[0].AsFeatureFlag()
				expected := evaluator.Evaluate(&flag, userID)

				body, err := json.Marshal(struct {
					UserID domain.UserID `json:"user_id"`
				}{UserID: userID})
				if err != nil {
					t.Fatal(err)
				}

				response := httpRequest(router, http.MethodPost, delivery.AdminBasePath+"/flags/1/evaluate", string(body), "admin", "")
				if response.Code != http.StatusOK {
					t.Fatal(response.Code, response.Body.String())
				}

				var decision oc.Decision
				if err := json.Unmarshal(response.Body.Bytes(), &decision); err != nil {
					t.Fatal(err)
				}

				if !reflect.DeepEqual(decision.Evaluation, expected) || decision.Revision != payload.Revision || decision.UserID != userID {
					t.Fatalf("evaluator mismatch: user=%d response=%+v", userID, decision)
				}
			}
		}
	}
}

func TestSharedSnapshotValidationRejectsCorruptRules(t *testing.T) {
	for _, alter := range []func(*oc.SnapshotPayload){
		func(p *oc.SnapshotPayload) { p.Revision = -1 },
		func(p *oc.SnapshotPayload) { p.Configs = append(p.Configs, p.Configs[0]) },
		func(p *oc.SnapshotPayload) { p.Flags[0].RolloutPercent = 101 },
		func(p *oc.SnapshotPayload) { p.Flags[0].Salt = "" },
		func(p *oc.SnapshotPayload) {
			p.Flags[0].ForceOffUserIDs = append(p.Flags[0].ForceOffUserIDs, p.Flags[0].ForceOnUserIDs[0])
		},
	} {
		payload := loadContractFixture(t, "snapshot_full.json")
		alter(&payload)

		if err := payload.Validate(); !errors.Is(err, oc.ErrInvalidSnapshot) {
			t.Fatal(err)
		}
	}
}
