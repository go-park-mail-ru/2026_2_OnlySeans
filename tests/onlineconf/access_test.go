package onlineconf_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/auth"
	delivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/onlineconf"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	oc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain/onlineconf"
	service "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/onlineconf"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/tests/testutil"
)

const (
	accessTestToken  = "test-service-token"
	accessTestOrigin = "https://product.example.test"
)

type sessionReaderFunc func(context.Context, string) (*domain.Session, error)

func (f sessionReaderFunc) Get(ctx context.Context, id string) (*domain.Session, error) {
	return f(ctx, id)
}

func TestOnlineConfAdminAccess(t *testing.T) {
	failure := errors.New("storage failure")

	cases := []struct {
		name       string
		session    *domain.Session
		sessionErr error
		user       *domain.User
		userErr    error
		want       error
	}{
		{name: "admin", session: &domain.Session{UserID: testActor}, user: &domain.User{ID: testActor, Username: "admin", Role: domain.RoleAdmin}},
		{name: "reader", session: &domain.Session{UserID: testActor}, user: &domain.User{ID: testActor, Role: domain.RoleReader}, want: domain.ErrForbidden},
		{name: "editor", session: &domain.Session{UserID: testActor}, user: &domain.User{ID: testActor, Role: domain.RoleEditor}, want: domain.ErrForbidden},
		{name: "missing session", sessionErr: domain.ErrSessionNotFound, want: domain.ErrUnauthorized},
		{name: "nil session", want: domain.ErrUnauthorized},
		{name: "expired", session: &domain.Session{UserID: testActor, ExpiresAt: time.Now().Add(-time.Hour)}, want: domain.ErrUnauthorized},
		{name: "deleted user", session: &domain.Session{UserID: testActor}, userErr: domain.ErrUserNotFound, want: domain.ErrUnauthorized},
		{name: "nil user", session: &domain.Session{UserID: testActor}, want: domain.ErrUnauthorized},
		{name: "redis failure", sessionErr: failure, want: failure},
		{name: "database failure", session: &domain.Session{UserID: testActor}, userErr: failure, want: failure},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sessions := sessionReaderFunc(func(context.Context, string) (*domain.Session, error) { return tc.session, tc.sessionErr })
			users := userReaderFunc(func(context.Context, domain.UserID) (*domain.User, error) { return tc.user, tc.userErr })

			access, err := service.NewAccess(sessions, users, testutil.DiscardLogger())
			if err != nil {
				t.Fatal(err)
			}

			actor, err := access.AuthenticateAdmin(context.Background(), "session")
			if !errors.Is(err, tc.want) {
				t.Fatalf("error=%v want=%v", err, tc.want)
			}

			if tc.want == nil && (actor.ID != testActor || actor.Role != domain.RoleAdmin) {
				t.Fatal(actor)
			}
		})
	}
}

func TestOnlineConfMiddlewareAdminAndCSRF(t *testing.T) {
	user := &domain.User{ID: testActor, Username: "admin", Role: domain.RoleAdmin}
	revoked := false
	failure := false
	sessions := sessionReaderFunc(func(_ context.Context, id string) (*domain.Session, error) {
		if failure {
			return nil, errors.New("redis unavailable")
		}

		if revoked || id != "valid" {
			return nil, domain.ErrSessionNotFound
		}

		return &domain.Session{UserID: testActor}, nil
	})
	users := userReaderFunc(func(context.Context, domain.UserID) (*domain.User, error) { return user, nil })

	access, err := service.NewAccess(sessions, users, testutil.DiscardLogger())
	if err != nil {
		t.Fatal(err)
	}

	middleware, err := delivery.NewMiddleware(access, accessTestToken, []string{accessTestOrigin}, testutil.DiscardLogger())
	if err != nil {
		t.Fatal(err)
	}

	handler := middleware.RequireAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actor, ok := delivery.AdminFromContext(r.Context())
		id, exists := auth.UserIDFromContext(r.Context())

		if !ok || !exists || actor.ID != testActor || id != testActor {
			t.Fatal("unverified actor")
		}

		w.WriteHeader(http.StatusNoContent)
	}))

	request := func(method, cookie, origin, site string) int {
		t.Helper()

		r := httptest.NewRequest(method, accessTestOrigin+"/api/onlineconf/admin/configs?user_id=999", strings.NewReader(`{"actor_id":999}`))
		r = r.WithContext(auth.ContextWithUserID(r.Context(), 999))
		r.Header.Set("X-Role", "admin")
		r.Header.Set("Origin", origin)
		r.Header.Set("Sec-Fetch-Site", site)

		if cookie != "" {
			r.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: cookie})
		}

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)

		if w.Header().Get("Cache-Control") != "private, no-store" || w.Header().Get("Set-Cookie") != "" {
			t.Fatal("cache or cookie changed")
		}

		if w.Code != http.StatusNoContent {
			var body struct {
				Error string `json:"error"`
				Code  string `json:"code"`
			}

			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || body.Code == "" {
				t.Fatalf("invalid API error: %s", w.Body.String())
			}
		}

		return w.Code
	}
	if status := request(http.MethodGet, "", "", ""); status != http.StatusUnauthorized {
		t.Fatal(status)
	}

	if status := request(http.MethodGet, "valid", "", ""); status != http.StatusNoContent {
		t.Fatal(status)
	}

	if status := request(http.MethodPost, "valid", "https://evil.example", "cross-site"); status != http.StatusForbidden {
		t.Fatal(status)
	}

	if status := request(http.MethodPost, "valid", accessTestOrigin, "same-origin"); status != http.StatusNoContent {
		t.Fatal(status)
	}

	user.Role = domain.RoleReader

	if status := request(http.MethodGet, "valid", "", ""); status != http.StatusForbidden {
		t.Fatal(status)
	}

	user.Role = domain.RoleAdmin
	revoked = true

	if status := request(http.MethodGet, "valid", "", ""); status != http.StatusUnauthorized {
		t.Fatal(status)
	}

	revoked = false
	failure = true

	if status := request(http.MethodGet, "valid", "", ""); status != http.StatusInternalServerError {
		t.Fatal(status)
	}
}

func TestOnlineConfMiddlewareServiceAuthorization(t *testing.T) {
	sessions := sessionReaderFunc(func(context.Context, string) (*domain.Session, error) {
		t.Fatal("service auth used session")
		return nil, nil
	})
	users := userReaderFunc(func(context.Context, domain.UserID) (*domain.User, error) {
		t.Fatal("service auth used user role")
		return nil, nil
	})

	access, err := service.NewAccess(sessions, users, testutil.DiscardLogger())
	if err != nil {
		t.Fatal(err)
	}

	middleware, err := delivery.NewMiddleware(access, accessTestToken, nil, testutil.DiscardLogger())
	if err != nil {
		t.Fatal(err)
	}

	handler := middleware.RequireService(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := delivery.AdminFromContext(r.Context()); ok {
			t.Fatal("service request acquired actor")
		}

		w.WriteHeader(http.StatusNoContent)
	}))

	for _, header := range []string{"", "Basic " + accessTestToken, "Bearer wrong", "Bearer", "Bearer " + accessTestToken + " extra", "Bearer " + accessTestToken} {
		r := httptest.NewRequest(http.MethodGet, "http://internal/internal/onlineconf/snapshot?user_id=1", nil)
		r.Header.Set("Authorization", header)
		r.Header.Set("X-Role", "admin")
		r.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "valid"})

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)

		expected := http.StatusUnauthorized
		if header == "Bearer "+accessTestToken {
			expected = http.StatusNoContent
		}

		if w.Code != expected {
			t.Fatalf("status=%d want=%d", w.Code, expected)
		}
	}

	for _, token := range []string{"", "  ", "secret token"} {
		if _, err := delivery.NewMiddleware(access, token, nil, testutil.DiscardLogger()); !errors.Is(err, oc.ErrInvalidServiceToken) {
			t.Fatal(err)
		}
	}
}
