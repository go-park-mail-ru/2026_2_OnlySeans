package router

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

const testOrigin = "http://127.0.0.1:5500"

func TestWithCORS(t *testing.T) {
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusTeapot)
	})
	handler := withCORS(next, testOrigin)

	tests := []struct {
		name            string
		method          string
		origin          string
		wantStatus      int
		wantNextCall    bool
		wantAllowOrigin string
	}{
		{"preflight отвечает middleware", http.MethodOptions, testOrigin, http.StatusNoContent, false, testOrigin},
		{"обычный запрос доходит до хендлера", http.MethodGet, testOrigin, http.StatusTeapot, true, testOrigin},
		{"чужой origin не получает разрешения", http.MethodGet, "http://evil.example", http.StatusTeapot, true, ""},
		{"запрос без origin", http.MethodGet, "", http.StatusTeapot, true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextCalled = false

			req := httptest.NewRequest(tt.method, "/", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("статус = %d, ожидали %d", rec.Code, tt.wantStatus)
			}
			if nextCalled != tt.wantNextCall {
				t.Errorf("next вызван = %v, ожидали %v", nextCalled, tt.wantNextCall)
			}
			if got := rec.Header().Get("Access-Control-Allow-Origin"); got != tt.wantAllowOrigin {
				t.Errorf("Allow-Origin = %q, ожидали %q", got, tt.wantAllowOrigin)
			}
			if tt.wantAllowOrigin != "" && rec.Header().Get("Access-Control-Allow-Credentials") != "true" {
				t.Error("нет Access-Control-Allow-Credentials: true")
			}
		})
	}
}
