package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func doRequest(t *testing.T, handlerFunc http.HandlerFunc, body interface{}) (int, map[string]interface{}) {
	t.Helper()

	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("не удалось замаршалить тело запроса: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handlerFunc(rec, req)

	var parsed map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &parsed)

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
				t.Errorf("статус = %d, ожидали %d (тело: %v)", status, tt.wantStatus, body)
			}
		})
	}
}

func newTestHandler(t *testing.T) *Handler {
	t.Helper()
	h, err := NewHandler(newTestUseCase(t))
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
		t.Fatalf("первая регистрация: статус = %d, ожидали %d", status, http.StatusCreated)
	}

	status, body := doRequest(t, h.Register, registerRequest{Email: "dup@example.com", Username: "second", Password: "Password1"})
	if status != http.StatusConflict {
		t.Errorf("повторная регистрация: статус = %d, ожидали %d (тело: %v)", status, http.StatusConflict, body)
	}
}

func TestHandler_Login(t *testing.T) {
	h := newTestHandler(t)
	status, _ := doRequest(t, h.Register, registerRequest{Email: "login@example.com", Username: "loginuser", Password: "Password1"})
	if status != http.StatusCreated {
		t.Fatalf("не удалось подготовить пользователя: статус %d", status)
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
				t.Errorf("статус = %d, ожидали %d (тело: %v)", status, tt.wantStatus, body)
			}
		})
	}
}

func TestNewHandler_NilUseCase(t *testing.T) {
	h, err := NewHandler(nil)
	if h != nil || err != ErrNilUseCase {
		t.Errorf("got (%v, %v), want (nil, %v)", h, err, ErrNilUseCase)
	}
}

func TestHandler_SessionFlow(t *testing.T) {
	h := newTestHandler(t)

	raw, err := json.Marshal(registerRequest{Email: "flow@example.com", Username: "flowuser", Password: "Password1"})
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	h.Register(rec, httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(raw)))
	if rec.Code != http.StatusCreated {
		t.Fatalf("регистрация: статус = %d", rec.Code)
	}

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != sessionCookieName || cookies[0].Value == "" {
		t.Fatalf("ожидали одну куку %q, получили %v", sessionCookieName, cookies)
	}
	cookie := cookies[0]
	if !cookie.HttpOnly {
		t.Error("кука должна быть HttpOnly")
	}

	callMe := func(c *http.Cookie) int {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		if c != nil {
			req.AddCookie(c)
		}
		rec := httptest.NewRecorder()
		h.RequireAuth(h.Me)(rec, req)
		return rec.Code
	}

	if got := callMe(nil); got != http.StatusUnauthorized {
		t.Errorf("/me без куки: статус = %d, ожидали %d", got, http.StatusUnauthorized)
	}
	if got := callMe(cookie); got != http.StatusOK {
		t.Errorf("/me с кукой: статус = %d, ожидали %d", got, http.StatusOK)
	}

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	h.Logout(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("logout: статус = %d, ожидали %d", rec.Code, http.StatusNoContent)
	}

	if got := callMe(cookie); got != http.StatusUnauthorized {
		t.Errorf("/me после logout: статус = %d, ожидали %d", got, http.StatusUnauthorized)
	}
}
