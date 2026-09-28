package app

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestWithCORS(t *testing.T) {
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusTeapot)
	})
	handler := withCORS(next)

	tests := []struct {
		name         string
		method       string
		wantStatus   int
		wantNextCall bool
	}{
		{
			name:         "preflight is answered by middleware",
			method:       http.MethodOptions,
			wantStatus:   http.StatusOK,
			wantNextCall: false,
		},
		{
			name:         "regular request reaches handler",
			method:       http.MethodGet,
			wantStatus:   http.StatusTeapot,
			wantNextCall: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextCalled = false
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(tt.method, "/api/login", nil))

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if nextCalled != tt.wantNextCall {
				t.Errorf("next called = %v, want %v", nextCalled, tt.wantNextCall)
			}
			if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
				t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, "*")
			}
		})
	}
}

func TestRun_InvalidConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("server:\n  port: 0\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := Run(path); err == nil {
		t.Error("expected error for invalid config")
	}
}

func TestRun_MissingConfig(t *testing.T) {
	if err := Run(filepath.Join(t.TempDir(), "missing.yaml")); err == nil {
		t.Error("expected error for missing config")
	}
}
