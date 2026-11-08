package router_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPosterRouteRequiresAdmin(t *testing.T) {
	tests := []struct {
		name          string
		authenticated bool
		wantStatus    int
	}{
		{
			name: "anonymous rejected", wantStatus: http.StatusUnauthorized,
		},
		{
			name: "reader rejected", authenticated: true, wantStatus: http.StatusForbidden,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := newTestRouter(t)
			req := httptest.NewRequest(http.MethodPost, "/api/films/1/poster", nil)
			if tt.authenticated {
				register := httptest.NewRecorder()
				handler.ServeHTTP(register, httptest.NewRequest(http.MethodPost, "/api/register", strings.NewReader(`{"email":"poster-reader@example.com","username":"posterreader","password":"Password1"}`)))
				require.Equal(t, http.StatusCreated, register.Code)
				req.AddCookie(register.Result().Cookies()[0])
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			require.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}
