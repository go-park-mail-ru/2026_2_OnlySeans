package router_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRequestRejectionLogs(t *testing.T) {
	tests := []struct {
		name    string
		method  string
		path    string
		body    string
		message string
		field   string
	}{
		{
			name:    "registration JSON",
			method:  http.MethodPost,
			path:    "/api/register",
			body:    `{"password":"private-test-value","unknown_field":true}`,
			message: "request rejected",
			field:   "body",
		},
		{
			name:    "registration password validation",
			method:  http.MethodPost,
			path:    "/api/register",
			body:    `{"email":"test@example.com","username":"tester","password":"x"}`,
			message: "registration rejected: invalid input",
			field:   "password",
		},
		{
			name:    "login JSON",
			method:  http.MethodPost,
			path:    "/api/login",
			body:    `{"password":"private-test-value",`,
			message: "request rejected",
			field:   "body",
		},
		{
			name:    "film ID",
			method:  http.MethodGet,
			path:    "/api/films/invalid",
			message: "request rejected",
			field:   "id",
		},
		{
			name:    "collection pagination",
			method:  http.MethodGet,
			path:    "/api/collections?limit=invalid",
			message: "request rejected",
			field:   "pagination",
		},
		{
			name:    "rating film ID",
			method:  http.MethodGet,
			path:    "/api/films/invalid/rating",
			message: "request rejected",
			field:   "id",
		},
		{
			name:    "rating collection ID",
			method:  http.MethodGet,
			path:    "/api/collections/invalid/ratings",
			message: "request rejected",
			field:   "id",
		},
		{
			name:    "rating pagination",
			method:  http.MethodGet,
			path:    "/api/collections/1/ratings?limit=invalid",
			message: "request rejected",
			field:   "pagination",
		},
		{
			name:    "person ID",
			method:  http.MethodGet,
			path:    "/api/persons/invalid",
			message: "request rejected",
			field:   "id",
		},
		{
			name:    "person pagination",
			method:  http.MethodGet,
			path:    "/api/persons?offset=invalid",
			message: "request rejected",
			field:   "pagination",
		},
		{
			name:    "reviews pagination",
			method:  http.MethodGet,
			path:    "/api/films/1/reviews?limit=invalid",
			message: "request rejected",
			field:   "pagination",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appLogger := newSpyLogger(t)
			handler := newTestRouterWithLogger(t, appLogger)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body)))
			require.Equal(t, http.StatusBadRequest, rec.Code)
			found := 0
			for _, entry := range appLogger.entries {
				require.NotEqual(t, "error", entry.level)
				if entry.msg == tt.message {
					found++
					require.Equal(t, "warn", entry.level)
					require.Equal(t, tt.field, entry.fields["field"])
					if tt.message == "request rejected" {
						require.Equal(t, tt.method, entry.fields["method"])
						require.NotEmpty(t, entry.fields["path"])
					}
				}
				for _, value := range entry.fields {
					require.NotContains(t, fmt.Sprint(value), "private-test-value")
				}
			}
			require.Equal(t, 1, found)
		})
	}
}
