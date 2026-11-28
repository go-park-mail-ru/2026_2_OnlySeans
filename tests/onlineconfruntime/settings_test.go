package onlineconfruntime_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	authdelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/auth"
	delivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/onlineconfruntime"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	oc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain/onlineconf"
	service "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/onlineconf"
	evaluator "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/onlineconfevaluator"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/tests/testutil"
)

func settingsPayload() oc.SnapshotPayload {
	return oc.SnapshotPayload{
		Revision: 1, GeneratedAt: time.Now().UTC(),
		Configs: []oc.SnapshotConfig{
			{Scope: "frontend", Key: "button_color", Value: json.RawMessage(`"red&blue"`)},
			{Scope: "frontend", Key: "layout", Value: json.RawMessage(`{"columns":3}`)},
			{Scope: "frontend", Key: "tabs", Value: json.RawMessage(`["films","series"]`)},
			{Scope: "payments", Key: "secret", Value: json.RawMessage(`"private-value"`)},
		},
		Flags: []oc.SnapshotFlag{
			{ID: 1, Scope: "frontend", Key: "new_navigation", Enabled: true, RolloutPercent: 25, Salt: "demo", ForceOnUserIDs: []domain.UserID{7}, ForceOffUserIDs: []domain.UserID{8}},
			{ID: 2, Scope: "payments", Key: "internal_mode", Enabled: true, RolloutPercent: 100, Salt: "private-salt"},
		},
	}
}

func TestGenericSettingsUseMemoryAndProtectSnapshot(t *testing.T) {
	raw := settingsPayload()
	next := settingsPayload()
	next.Revision++
	next.Configs[0].Value = json.RawMessage(`"green"`)
	source := &sequenceSource{results: []sourceResult{{snapshot: snapshotFromPayload(t, raw)}, {snapshot: snapshotFromPayload(t, next)}}}
	cache, err := service.NewCache(source, time.Second, testutil.DiscardLogger())
	require.NoError(t, err)
	var settings service.SettingsProvider = cache
	_, found := settings.GetConfig("frontend", "button_color")
	require.False(t, found)
	require.False(t, settings.EvaluateFlag("frontend", "new_navigation", 7).Enabled)
	require.NoError(t, cache.Refresh(t.Context()))

	for range 100 {
		value, ok := settings.GetConfig("payments", "secret")
		require.True(t, ok)
		require.JSONEq(t, `"private-value"`, string(value))
		value[0] = 'X'

		require.True(t, settings.EvaluateFlag("frontend", "new_navigation", 7).Enabled)
	}

	require.Equal(t, 1, source.callCount())

	for _, userID := range []domain.UserID{0, 1, 3, 7, 8, 42, 100, domain.UserID(oc.MaxSafeID), domain.UserID(oc.MaxSafeID + 1)} {
		flag := raw.Flags[0].AsFeatureFlag()
		require.Equal(t, evaluator.Evaluate(&flag, userID), settings.EvaluateFlag("frontend", "new_navigation", userID))
	}

	_, found = settings.GetConfig("frontend", "absent")
	require.False(t, found)
	require.False(t, settings.EvaluateFlag("frontend", "absent", 7).Enabled)

	view := cache.SettingsSnapshot()
	require.NoError(t, cache.Refresh(t.Context()))

	previous, found := view.GetConfig("frontend", "button_color")
	require.True(t, found)
	require.JSONEq(t, `"red&blue"`, string(previous))

	current, found := cache.GetConfig("frontend", "button_color")
	require.True(t, found)
	require.JSONEq(t, `"green"`, string(current))
	require.Equal(t, raw.Revision, view.Revision())
	require.Equal(t, next.Revision, cache.SettingsSnapshot().Revision())
}

func TestRuntimePublishesFrontendScopeWithoutPrivateResources(t *testing.T) {
	source := &sequenceSource{results: []sourceResult{{snapshot: snapshotFromPayload(t, settingsPayload())}}}
	cache, err := service.NewCache(source, time.Second, testutil.DiscardLogger())
	require.NoError(t, err)
	require.NoError(t, cache.Refresh(t.Context()))
	handler, err := delivery.NewHandler(cache, testutil.DiscardLogger())
	require.NoError(t, err)

	for _, userID := range []domain.UserID{7, 8} {
		req := httptest.NewRequest(http.MethodGet, "/api/onlineconf/runtime?user_id=7", nil)
		req = req.WithContext(authdelivery.ContextWithUserID(req.Context(), userID))
		rec := httptest.NewRecorder()
		handler.Runtime(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)
		var body struct {
			Configs map[string]json.RawMessage `json:"configs"`
			Flags   map[string]bool            `json:"flags"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.JSONEq(t, `"red&blue"`, string(body.Configs["frontend.button_color"]))
		require.JSONEq(t, `{"columns":3}`, string(body.Configs["frontend.layout"]))
		require.JSONEq(t, `["films","series"]`, string(body.Configs["frontend.tabs"]))
		require.Equal(t, userID == 7, body.Flags["frontend.new_navigation"])

		for _, secret := range []string{"payments", "private-value", "private-salt", "force_on_user_ids", "force_off_user_ids"} {
			require.NotContains(t, rec.Body.String(), secret)
		}
	}
}

func TestInvalidSnapshotPreservesLastSuccessfulState(t *testing.T) {
	good := settingsPayload()

	for name, alter := range map[string]func(*oc.SnapshotPayload){
		"conflicting overrides": func(p *oc.SnapshotPayload) { p.Flags[0].ForceOffUserIDs = []domain.UserID{7} },
		"invalid flag ID":       func(p *oc.SnapshotPayload) { p.Flags[0].ID = 0 },
		"duplicate flag ID":     func(p *oc.SnapshotPayload) { p.Flags[1].ID = p.Flags[0].ID },
		"invalid scope":         func(p *oc.SnapshotPayload) { p.Configs[0].Scope = "invalid.scope" },
	} {
		t.Run(name, func(t *testing.T) {
			payload := settingsPayload()
			var published atomic.Pointer[oc.SnapshotPayload]
			published.Store(&payload)

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				require.NoError(t, json.NewEncoder(w).Encode(published.Load()))
			}))
			defer server.Close()
			client, err := service.NewHTTPClient(server.Client(), server.URL, oc.SnapshotAPIPath, "test-token")
			require.NoError(t, err)
			cache, err := service.NewCache(client, time.Second, testutil.DiscardLogger())
			require.NoError(t, err)
			require.NoError(t, cache.Refresh(t.Context()))
			view := cache.SettingsSnapshot()
			invalid := settingsPayload()
			invalid.Revision++
			alter(&invalid)
			published.Store(&invalid)
			require.Error(t, cache.Refresh(t.Context()))
			require.Equal(t, good.Revision, cache.Status().Revision)
			require.Same(t, view, cache.SettingsSnapshot())
		})
	}
}
