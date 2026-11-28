package onlineconf

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	authdelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/tests/testutil"
	"github.com/stretchr/testify/require"
)

type capturingPolicy struct {
	userID domain.UserID
}

func (p *capturingPolicy) ReviewPolicy(userID domain.UserID) domain.ReviewPolicy {
	p.userID = userID
	return domain.ReviewPolicy{WriteEnabled: userID == 7, MaxLength: 321, Revision: 8}
}

func TestRuntimeUsesSessionContextAndOnlyPublicProjection(t *testing.T) {
	policy := &capturingPolicy{}
	handler, err := NewHandler(policy, testutil.DiscardLogger())
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodGet, "/api/onlineconf/runtime?user_id=999", nil)
	req = req.WithContext(authdelivery.ContextWithUserID(req.Context(), 7))
	rec := httptest.NewRecorder()

	handler.Runtime(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "private, no-store", rec.Header().Get("Cache-Control"))
	require.Equal(t, "Cookie", rec.Header().Get("Vary"))
	require.Equal(t, domain.UserID(7), policy.userID, "query user_id must not affect evaluation")
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, float64(8), body["revision"])
	require.Equal(t, map[string]any{domain.ReviewMaxLengthKey: float64(321)}, body["configs"])
	require.Equal(t, map[string]any{domain.ReviewWriteEnabledKey: true}, body["flags"])
	require.NotContains(t, rec.Body.String(), "included_user_ids")
	require.NotContains(t, rec.Body.String(), "token")
}
