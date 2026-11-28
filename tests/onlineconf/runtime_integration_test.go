package onlineconf_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/auth"
	runtime "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/onlineconfruntime"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	oc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain/onlineconf"
	service "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/onlineconf"
	reviews "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/reviews"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/tests/mocks"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/tests/testutil"
)

const runtimeIntegrationInterval = 5 * time.Millisecond

func TestAdminHomeBlocksWritesAndLegacyRecovery(t *testing.T) {
	repo, db := newRepository(t)
	router := newHTTPRouter(t, repo, &snapshotStub{value: snapshotFixture(1)})
	response := httpRequest(router, http.MethodPost, adminAPI+"/scopes", `{"name":"frontend","title":"Frontend"}`, "admin", "")
	require.Equal(t, http.StatusCreated, response.Code)

	const createBody = `{"scope":"frontend","key":"home_blocks","title":"Home","value":%s}`
	const updateBody = `{"scope":"frontend","key":"home_blocks","title":"Home","description":"","value":%s,"version":1}`
	const invalid = `[{"type":"banner"}]`
	const valid = `[{"type":"collection","id":1}]`
	configs := adminAPI + "/configs"
	response = httpRequest(router, http.MethodPost, configs, fmt.Sprintf(createBody, invalid), "admin", "")
	require.Equal(t, http.StatusBadRequest, response.Code)
	require.Contains(t, response.Body.String(), "invalid_input")
	require.Contains(t, response.Body.String(), "title")
	response = httpRequest(router, http.MethodPost, configs, fmt.Sprintf(createBody, valid), "admin", "")
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
	var config oc.Config
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &config))
	resource := fmt.Sprintf("%s/%d", configs, config.ID)
	response = httpRequest(router, http.MethodPut, resource, fmt.Sprintf(updateBody, invalid), "admin", "")
	require.Equal(t, http.StatusBadRequest, response.Code)
	require.Contains(t, response.Body.String(), "block 1: title")
	saved, err := repo.GetConfig(t.Context(), config.ID)
	require.NoError(t, err)
	require.Equal(t, config.Version, saved.Version)
	require.JSONEq(t, string(config.Value), string(saved.Value))

	// Rows written before schema validation must not block other commits or repair.
	_, err = db.ExecContext(t.Context(), `UPDATE oc_configs SET value='[{"type":"banner"}]' WHERE id=$1`, config.ID)
	require.NoError(t, err)
	response = httpRequest(router, http.MethodPost, configs, `{"scope":"frontend","key":"color","title":"Color","value":"red"}`, "admin", "")
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
	response = httpRequest(router, http.MethodPut, resource, fmt.Sprintf(updateBody, valid), "admin", "")
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
}

func TestAdminCommitReachesProductRuntimeAndReviewUseCase(t *testing.T) {
	repo, _ := newRepository(t)
	appLogger := testutil.DiscardLogger()
	ctx, cancel := context.WithCancel(t.Context())
	store, err := service.NewSnapshotStore(repo, service.SnapshotOptions{RefreshInterval: runtimeIntegrationInterval}, appLogger)
	require.NoError(t, err)
	require.NoError(t, store.Start(ctx))
	t.Cleanup(func() {
		cancel()
		<-store.Done()
	})

	router := newHTTPRouterWithRefresher(t, repo, store, store)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	client, err := service.NewHTTPClient(server.Client(), server.URL, oc.SnapshotAPIPath, accessTestToken)
	require.NoError(t, err)
	cache, err := service.NewCache(client, runtimeIntegrationInterval, appLogger)
	require.NoError(t, err)
	require.NoError(t, cache.Refresh(ctx))

	done := make(chan struct{})

	go func() {
		defer close(done)
		cache.Run(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})

	for _, scope := range []string{"reviews", "frontend", "payments"} {
		response := httpRequest(router, http.MethodPost, adminAPI+"/scopes", fmt.Sprintf(`{"name":%q,"title":%q}`, scope, scope), "admin", "")
		require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
	}

	for _, body := range []string{
		`{"scope":"reviews","key":"max_length","title":"Limit","value":50}`,
		`{"scope":"frontend","key":"button_color","title":"Color","value":"red&blue"}`,
		`{"scope":"payments","key":"secret","title":"Secret","value":"private-value"}`,
	} {
		response := httpRequest(router, http.MethodPost, adminAPI+"/configs", body, "admin", "")
		require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
	}

	response := httpRequest(router, http.MethodPost, adminAPI+"/flags", `{"scope":"reviews","key":"write_enabled","title":"Reviews","enabled":true,"rollout_percent":0,"force_on_user_ids":[2],"force_off_user_ids":[]}`, "admin", "")
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
	var flag oc.FeatureFlag
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &flag))
	response = httpRequest(router, http.MethodPost, adminAPI+"/flags", `{"scope":"frontend","key":"new_navigation","title":"Navigation","enabled":true,"rollout_percent":100,"force_on_user_ids":[],"force_off_user_ids":[]}`, "admin", "")
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
	require.Eventually(t, func() bool { return cache.ReviewPolicy(testOtherUser).Revision == 8 }, time.Second, runtimeIntegrationInterval)

	response = httpRequest(router, http.MethodPost, fmt.Sprintf("%s/flags/%d/evaluate", adminAPI, flag.ID), `{"user_id":2}`, "admin", "")
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	var decision oc.Decision
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &decision))
	require.Equal(t, decision.Evaluation, cache.EvaluateFlag("reviews", "write_enabled", testOtherUser))
	require.Equal(t, decision.Revision, cache.ReviewPolicy(testOtherUser).Revision)

	handler, err := runtime.NewHandler(cache, appLogger)
	require.NoError(t, err)

	request := httptest.NewRequest(http.MethodGet, "/api/onlineconf/runtime?user_id=1", nil)
	request = request.WithContext(auth.ContextWithUserID(request.Context(), testOtherUser))
	recorder := httptest.NewRecorder()
	handler.Runtime(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "private, no-store", recorder.Header().Get("Cache-Control"))
	var body struct {
		Revision int64                      `json:"revision"`
		Configs  map[string]json.RawMessage `json:"configs"`
		Flags    map[string]bool            `json:"flags"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Equal(t, decision.Revision, body.Revision)
	require.JSONEq(t, `"red&blue"`, string(body.Configs["frontend.button_color"]))
	require.JSONEq(t, `50`, string(body.Configs[domain.ReviewMaxLengthKey]))
	require.True(t, body.Flags[domain.ReviewWriteEnabledKey])
	require.True(t, body.Flags["frontend.new_navigation"])
	require.NotContains(t, recorder.Body.String(), "payments")
	require.NotContains(t, recorder.Body.String(), "force_on_user_ids")

	controller := gomock.NewController(t)
	create, err := reviews.NewCreateReviewUseCase(mocks.NewMockReviewRepository(controller), mocks.NewMockFilmRepository(controller), cache, appLogger)
	require.NoError(t, err)
	_, err = create.Run(ctx, reviews.NewCreateReviewInput(testOtherUser, 1, domain.ReviewInput{Title: "Review", Content: strings.Repeat("a", 51)}))
	var tooLong *domain.ReviewTooLongError
	require.ErrorAs(t, err, &tooLong)
	require.Equal(t, body.Revision, tooLong.Revision)
	require.Equal(t, 50, tooLong.MaxLength)

	response = httpRequest(router, http.MethodPut, fmt.Sprintf("%s/flags/%d", adminAPI, flag.ID), `{"scope":"reviews","key":"write_enabled","title":"Reviews","description":"","enabled":false,"rollout_percent":100,"force_on_user_ids":[2],"force_off_user_ids":[],"version":1}`, "admin", "")
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	require.Eventually(t, func() bool { return cache.ReviewPolicy(testOtherUser).Revision == 9 }, time.Second, runtimeIntegrationInterval)

	_, err = create.Run(ctx, reviews.NewCreateReviewInput(testOtherUser, 1, domain.ReviewInput{Title: "Review", Content: strings.Repeat("a", 50)}))
	require.ErrorIs(t, err, domain.ErrFeatureDisabled)
}
