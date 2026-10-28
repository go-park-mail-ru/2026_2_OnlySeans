package auth_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/auth"
	authservice "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/auth"
	usecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/tests/fakes"
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
		{"пустой email", "", "someuser", "Password1", http.StatusBadRequest},
		{"короткое имя", "user3@example.com", "a", "Password1", http.StatusBadRequest},
		{"слабый пароль", "user2@example.com", "someuser", "weak", http.StatusBadRequest},
		{"пароль без цифры", "user4@example.com", "someuser", "Password", http.StatusBadRequest},
		{"слишком длинный пароль", "user5@example.com", "someuser", "Password1Password1Password1Pass12", http.StatusBadRequest},
		{"пароль со смайликом", "user6@example.com", "someuser", "Pass😊word1", http.StatusBadRequest},
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

	h, _, _ := newTestAuth(t)
	return h
}

func newTestAuth(t *testing.T) (*auth.Handler, *authservice.InMemoryUserRepo, *fakes.SessionStore) {
	t.Helper()

	users := authservice.NewInMemoryUserRepo()
	sessions := fakes.NewSessionStore()
	passwordService := authservice.BcryptPasswordService{}

	register, err := usecase.NewRegisterUserUseCase(users, sessions, passwordService)
	require.NoError(t, err)
	login, err := usecase.NewLoginUserUseCase(users, sessions, passwordService)
	require.NoError(t, err)
	logout, err := usecase.NewLogoutUserUseCase(sessions)
	require.NoError(t, err)
	authenticate, err := usecase.NewAuthenticateUserUseCase(sessions)
	require.NoError(t, err)
	getUser, err := usecase.NewGetUserUseCase(users)
	require.NoError(t, err)

	h, err := auth.NewHandler(auth.HandlerUseCases{
		Register:     register,
		Login:        login,
		Logout:       logout,
		Authenticate: authenticate,
		GetUser:      getUser,
	})
	require.NoError(t, err)
	return h, users, sessions
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
		{"некорректный email", "not-an-email", "Password1", http.StatusBadRequest},
		{"пустой пароль", "login@example.com", "", http.StatusBadRequest},
		{"длинный пароль не отклоняется валидацией логина", "login@example.com", "Password1Password1Password1Pass12", http.StatusUnauthorized},
		{"пробел в пароле не отклоняется валидацией логина", "login@example.com", "Pass word1", http.StatusUnauthorized},
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

func TestNewHandler_NilDependencies(t *testing.T) {
	h, err := auth.NewHandler(auth.HandlerUseCases{})
	assert.Nil(t, h)
	assert.EqualError(t, err, "register use case is nil")
}
