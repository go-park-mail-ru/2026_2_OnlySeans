package auth_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/auth"
)

type registerRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func doRequest(t *testing.T, handlerFunc http.HandlerFunc, body any) (int, map[string]any) {
	t.Helper()

	raw, err := json.Marshal(body)
	require.NoError(t, err, "failed to marshal request body")

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handlerFunc(rec, req)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &parsed),
		"failed to parse response body %q", rec.Body.String())

	return rec.Code, parsed
}

func TestHandler_Register(t *testing.T) {
	tests := []struct {
		name       string
		email      string
		username   string
		password   string
		wantStatus int
	}{
		{"успешная регистрация", "new@example.com", "newuser", "Password1", http.StatusCreated},
		{"некорректный email", "not-an-email", "someuser", "Password1", http.StatusBadRequest},
		{"слабый пароль", "user2@example.com", "someuser", "weak", http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestHandler(t)

			status, body := doRequest(t, h.Register, registerRequest{
				Email: tt.email, Username: tt.username, Password: tt.password,
			})
			if status != tt.wantStatus {
				t.Errorf("status = %d, want %d (body: %v)", status, tt.wantStatus, body)
			}
		})
	}
}

func newTestHandler(t *testing.T) *auth.Handler {
	t.Helper()
	h, err := auth.NewHandler(newTestUseCase(t))
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func TestHandler_Register_DuplicateEmail(t *testing.T) {
	h := newTestHandler(t)
	req := registerRequest{Email: "dup@example.com", Username: "first", Password: "Password1"}

	status, _ := doRequest(t, h.Register, req)
	if status != http.StatusCreated {
		t.Fatalf("first registration: status = %d, want %d", status, http.StatusCreated)
	}

	status, body := doRequest(t, h.Register, registerRequest{Email: "dup@example.com", Username: "second", Password: "Password1"})
	if status != http.StatusConflict {
		t.Errorf("duplicate registration: status = %d, want %d (body: %v)", status, http.StatusConflict, body)
	}
}

func TestHandler_Login(t *testing.T) {
	h := newTestHandler(t)
	status, _ := doRequest(t, h.Register, registerRequest{Email: "login@example.com", Username: "loginuser", Password: "Password1"})
	if status != http.StatusCreated {
		t.Fatalf("failed to set up user: status %d", status)
	}

	tests := []struct {
		name       string
		email      string
		password   string
		wantStatus int
	}{
		{"верные данные", "login@example.com", "Password1", http.StatusOK},
		{"неверный пароль", "login@example.com", "WrongPass1", http.StatusUnauthorized},
		{"несуществующий email", "ghost@example.com", "Password1", http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body := doRequest(t, h.Login, loginRequest{Email: tt.email, Password: tt.password})
			if status != tt.wantStatus {
				t.Errorf("status = %d, want %d (body: %v)", status, tt.wantStatus, body)
			}
		})
	}
}

func TestNewHandler_NilUseCase(t *testing.T) {
	h, err := auth.NewHandler(nil)
	if h != nil || err != auth.ErrNilUseCase {
		t.Errorf("got (%v, %v), want (nil, %v)", h, err, auth.ErrNilUseCase)
	}
}
