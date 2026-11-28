package onlineconf_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	evaluator "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/onlineconfevaluator"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/auth"
	delivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/onlineconf"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	oc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain/onlineconf"
	service "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/onlineconf"
	usecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/onlineconf"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/tests/testutil"
)

const adminAPI = delivery.AdminBasePath

type handlerRepository struct {
	oc.Repository
	createConfig func(context.Context, domain.UserID, oc.Config) (*oc.Config, error)
	getFlag      func(context.Context, int64) (*oc.FeatureFlag, error)
	listConfigs  func(context.Context, oc.ListOptions) ([]oc.Config, int, error)
}

func (r *handlerRepository) CreateConfig(ctx context.Context, actor domain.UserID, value oc.Config) (*oc.Config, error) {
	return r.createConfig(ctx, actor, value)
}
func (r *handlerRepository) GetFlag(ctx context.Context, id int64) (*oc.FeatureFlag, error) {
	return r.getFlag(ctx, id)
}
func (r *handlerRepository) ListConfigs(ctx context.Context, options oc.ListOptions) ([]oc.Config, int, error) {
	return r.listConfigs(ctx, options)
}

func newHTTPRouter(t *testing.T, repo oc.Repository, snapshots usecase.SnapshotProvider) http.Handler {
	t.Helper()

	appLogger := testutil.DiscardLogger()
	refresher := &refreshSpy{}

	scopes, err := usecase.NewScopesUseCase(repo, refresher, appLogger)
	if err != nil {
		t.Fatal(err)
	}

	configs, err := usecase.NewConfigsUseCase(repo, refresher, appLogger)
	if err != nil {
		t.Fatal(err)
	}

	flags, err := usecase.NewFlagsUseCase(repo, refresher, appLogger)
	if err != nil {
		t.Fatal(err)
	}

	users := userReaderFunc(func(_ context.Context, id domain.UserID) (*domain.User, error) {
		role := domain.RoleReader
		if id == testActor {
			role = domain.RoleAdmin
		}

		return &domain.User{ID: id, Username: "admin", Role: role}, nil
	})

	evaluate, err := usecase.NewEvaluateFlagUseCase(users, snapshots, evaluator.Evaluate, appLogger)
	if err != nil {
		t.Fatal(err)
	}

	audit, err := usecase.NewListAuditUseCase(repo, appLogger)
	if err != nil {
		t.Fatal(err)
	}

	sessions := sessionReaderFunc(func(_ context.Context, id string) (*domain.Session, error) {
		switch id {
		case "admin":
			return &domain.Session{UserID: testActor}, nil
		case "reader":
			return &domain.Session{UserID: testOtherUser}, nil
		default:
			return nil, domain.ErrSessionNotFound
		}
	})

	access, err := service.NewAccess(sessions, users, appLogger)
	if err != nil {
		t.Fatal(err)
	}

	middleware, err := delivery.NewMiddleware(access, accessTestToken, []string{accessTestOrigin}, appLogger)
	if err != nil {
		t.Fatal(err)
	}

	router, err := delivery.NewRouter(delivery.HandlerUseCases{Scopes: scopes, Configs: configs, Flags: flags, Evaluate: evaluate, Audit: audit}, middleware, snapshots, appLogger)
	if err != nil {
		t.Fatal(err)
	}

	return router
}

func httpRequest(router http.Handler, method, path, body, session, token string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, accessTestOrigin+path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")

	if session != "" {
		r.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: session})
	}

	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	return w
}

func TestOnlineConfHTTPMissingAndExplicitJSONValues(t *testing.T) {
	calls := 0
	repo := &handlerRepository{createConfig: func(_ context.Context, actor domain.UserID, value oc.Config) (*oc.Config, error) {
		calls++
		if actor != testActor {
			t.Fatal("unverified actor")
		}
		value.ID = 1
		value.Version = 1
		return &value, nil
	}}
	router := newHTTPRouter(t, repo, &snapshotStub{value: snapshotFixture(1)})

	for _, value := range []string{`null`, `false`, `0`, `"blue"`, `[]`, `{}`} {
		body := fmt.Sprintf(`{"scope":"frontend","key":"setting","title":"Setting","value":%s}`, value)

		response := httpRequest(router, http.MethodPost, adminAPI+"/configs", body, "admin", "")
		if response.Code != http.StatusCreated {
			t.Fatalf("%s: %d %s", value, response.Code, response.Body.String())
		}
	}

	previous := calls

	for _, body := range []string{
		`{"scope":"frontend","key":"setting","title":"Setting"}`,
		`{"scope":"frontend","key":"setting","title":"Setting","value":null,"actor_id":999}`,
		`{"scope":"frontend","key":"setting","title":"Setting","value":0,"version":1}`,
		`{"scope":"frontend","key":"setting","title":"Setting","value":0} {}`,
		`null`,
		`[]`,
		`{"scope":"frontend","key":"setting","title":"Setting","value":"` + strings.Repeat("x", delivery.MaxRequestBodyBytes) + `"}`,
	} {
		response := httpRequest(router, http.MethodPost, adminAPI+"/configs", body, "admin", "")
		if response.Code != http.StatusBadRequest {
			t.Fatalf("invalid request: %d %s", response.Code, response.Body.String())
		}
	}

	if calls != previous {
		t.Fatal("invalid request reached repository")
	}
}

func TestOnlineConfHTTPProtectsAllAdminPaths(t *testing.T) {
	router := newHTTPRouter(t, &handlerRepository{}, &snapshotStub{value: snapshotFixture(1)})
	for _, path := range []string{"/me", "/scopes", "/configs", "/configs/1", "/flags", "/flags/1", "/audit", "/unknown"} {
		if response := httpRequest(router, http.MethodGet, adminAPI+path, "", "", ""); response.Code != http.StatusUnauthorized {
			t.Fatalf("unguarded %s: %d", path, response.Code)
		}

		if response := httpRequest(router, http.MethodGet, adminAPI+path, "", "reader", ""); response.Code != http.StatusForbidden {
			t.Fatalf("reader access %s: %d", path, response.Code)
		}
	}

	if response := httpRequest(router, http.MethodPatch, adminAPI+"/configs", "", "admin", ""); response.Code != http.StatusMethodNotAllowed {
		t.Fatal(response.Code)
	}

	if response := httpRequest(router, http.MethodGet, adminAPI+"/unknown", "", "admin", ""); response.Code != http.StatusNotFound {
		t.Fatal(response.Code)
	}

	if response := httpRequest(router, http.MethodGet, adminAPI+"/me", "", "admin", ""); response.Code != http.StatusOK {
		t.Fatal(response.Code)
	}
}

func TestOnlineConfHTTPErrorCodesAndProjection(t *testing.T) {
	repo := &handlerRepository{createConfig: func(context.Context, domain.UserID, oc.Config) (*oc.Config, error) {
		return nil, fmt.Errorf("database wrapper: %w", oc.ErrVersionConflict)
	}, getFlag: func(context.Context, int64) (*oc.FeatureFlag, error) {
		return &oc.FeatureFlag{ID: 1, Scope: "frontend", Key: "test", Title: "Test", Salt: "private-salt"}, nil
	}}
	snapshots := &snapshotStub{value: snapshotFixture(1)}
	router := newHTTPRouter(t, repo, snapshots)
	response := httpRequest(router, http.MethodPost, adminAPI+"/configs", `{"scope":"frontend","key":"setting","title":"Setting","value":null}`, "admin", "")
	var errorBody struct {
		Code string `json:"code"`
	}

	if err := json.Unmarshal(response.Body.Bytes(), &errorBody); err != nil {
		t.Fatal(err)
	}

	if response.Code != http.StatusConflict || errorBody.Code != "version_conflict" {
		t.Fatal(response.Code, response.Body.String())
	}

	response = httpRequest(router, http.MethodGet, adminAPI+"/flags/1", "", "admin", "")
	if response.Code != http.StatusOK || strings.Contains(response.Body.String(), "salt") {
		t.Fatal(response.Code, response.Body.String())
	}

	response = httpRequest(router, http.MethodGet, delivery.InternalBasePath+"/snapshot", "", "admin", "")
	if response.Code != http.StatusUnauthorized {
		t.Fatal(response.Code)
	}

	response = httpRequest(router, http.MethodGet, delivery.InternalBasePath+"/snapshot", "", "", accessTestToken)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"salt":"stable"`) || strings.Contains(response.Body.String(), accessTestToken) {
		t.Fatal(response.Code, response.Body.String())
	}
}

func TestOnlineConfHTTPRequiredFlagFieldsAndPagination(t *testing.T) {
	repo := &handlerRepository{listConfigs: func(_ context.Context, options oc.ListOptions) ([]oc.Config, int, error) {
		if options.Scope != "frontend" || options.Query != "blue" || options.Limit != 1 || options.Offset != 2 {
			t.Fatal(options)
		}
		return nil, 0, nil
	}}

	router := newHTTPRouter(t, repo, &snapshotStub{value: snapshotFixture(1)})
	for _, body := range []string{
		`{"scope":"frontend","key":"flag","title":"Flag","rollout_percent":0,"force_on_user_ids":[],"force_off_user_ids":[]}`,
		`{"scope":"frontend","key":"flag","title":"Flag","enabled":false,"force_on_user_ids":[],"force_off_user_ids":[]}`,
		`{"scope":"frontend","key":"flag","title":"Flag","enabled":false,"rollout_percent":0,"force_on_user_ids":null,"force_off_user_ids":[]}`,
	} {
		response := httpRequest(router, http.MethodPost, adminAPI+"/flags", body, "admin", "")
		if response.Code != http.StatusBadRequest {
			t.Fatal(response.Code, response.Body.String())
		}
	}

	response := httpRequest(router, http.MethodGet, adminAPI+"/configs?scope=frontend&q=blue&limit=1&offset=2", "", "admin", "")
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"items":[]`) {
		t.Fatal(response.Code, response.Body.String())
	}

	for _, query := range []string{"limit=0", "limit=101", "offset=-1", "limit=oops"} {
		response = httpRequest(router, http.MethodGet, adminAPI+"/configs?"+query, "", "admin", "")
		if response.Code != http.StatusBadRequest {
			t.Fatal(response.Code, response.Body.String())
		}
	}
}

func TestOnlineConfHTTPPostgresWorkflow(t *testing.T) {
	repo, _ := newRepository(t)

	initial, err := repo.LoadSnapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	snapshots := &snapshotStub{value: initial}
	router := newHTTPRouter(t, repo, snapshots)

	response := httpRequest(router, http.MethodPost, adminAPI+"/scopes", `{"name":"reviews","title":"Отзывы"}`, "admin", "")
	if response.Code != http.StatusCreated {
		t.Fatal(response.Code, response.Body.String())
	}

	response = httpRequest(router, http.MethodPost, adminAPI+"/configs", `{"scope":"reviews","key":"max_length","title":"Лимит","value":500}`, "admin", "")
	if response.Code != http.StatusCreated {
		t.Fatal(response.Code, response.Body.String())
	}

	var config oc.Config
	if err := json.Unmarshal(response.Body.Bytes(), &config); err != nil {
		t.Fatal(err)
	}

	path := fmt.Sprintf("%s/configs/%d", adminAPI, config.ID)
	update := `{"scope":"reviews","key":"max_length","title":"Лимит","description":"","value":200,"version":1}`

	response = httpRequest(router, http.MethodPut, path, update, "admin", "")
	if response.Code != http.StatusOK {
		t.Fatal(response.Code, response.Body.String())
	}

	response = httpRequest(router, http.MethodPut, path, update, "admin", "")
	if response.Code != http.StatusConflict {
		t.Fatal(response.Code, response.Body.String())
	}

	for _, body := range []string{
		`{"scope":"reviews","key":"max_length","title":"Лимит","description":"","value":200}`,
		`{"scope":"reviews","key":"max_length","title":"Лимит","value":200,"version":2}`,
	} {
		response = httpRequest(router, http.MethodPut, path, body, "admin", "")
		if response.Code != http.StatusBadRequest {
			t.Fatal(response.Code, response.Body.String())
		}
	}

	response = httpRequest(router, http.MethodPost, adminAPI+"/flags", `{"scope":"reviews","key":"write_enabled","title":"Отзывы","enabled":true,"rollout_percent":0,"force_on_user_ids":[2],"force_off_user_ids":[]}`, "admin", "")
	if response.Code != http.StatusCreated || strings.Contains(response.Body.String(), `"salt"`) {
		t.Fatal(response.Code, response.Body.String())
	}

	var flag oc.FeatureFlag
	if err := json.Unmarshal(response.Body.Bytes(), &flag); err != nil {
		t.Fatal(err)
	}

	snapshots.value, err = repo.LoadSnapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	response = httpRequest(router, http.MethodPost, fmt.Sprintf("%s/flags/%d/evaluate", adminAPI, flag.ID), `{"user_id":2}`, "admin", "")
	var decision oc.Decision

	if err := json.Unmarshal(response.Body.Bytes(), &decision); err != nil {
		t.Fatal(err)
	}

	if response.Code != http.StatusOK || !decision.Enabled || decision.Reason != oc.ReasonForceOn || decision.Revision != 4 {
		t.Fatal(response.Code, response.Body.String())
	}

	response = httpRequest(router, http.MethodGet, adminAPI+"/audit?resource_type=config&resource_id="+fmt.Sprint(config.ID), "", "admin", "")
	var history usecase.ListOutput[oc.AuditEntry]

	if err := json.Unmarshal(response.Body.Bytes(), &history); err != nil {
		t.Fatal(err)
	}

	if response.Code != http.StatusOK || history.Total != 2 || history.Items[0].ActorID != testActor {
		t.Fatal(response.Code, response.Body.String())
	}

	response = httpRequest(router, http.MethodDelete, fmt.Sprintf("%s/flags/%d?version=1", adminAPI, flag.ID), "", "admin", "")
	if response.Code != http.StatusNoContent {
		t.Fatal(response.Code, response.Body.String())
	}

	snapshots.value, err = repo.LoadSnapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	response = httpRequest(router, http.MethodGet, delivery.InternalBasePath+"/snapshot", "", "", accessTestToken)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"revision":5`) || !strings.Contains(response.Body.String(), `"flags":[]`) {
		t.Fatal(response.Code, response.Body.String())
	}
}
