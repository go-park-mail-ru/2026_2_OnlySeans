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
			h := NewHandler(NewUseCase(NewInMemoryUserRepo(), nil))

			status, body := doRequest(t, h.Register, registerRequest{
				Email: tt.email, Username: tt.username, Password: tt.password,
			})
			if status != tt.wantStatus {
				t.Errorf("статус = %d, ожидали %d (тело: %v)", status, tt.wantStatus, body)
			}
		})
	}
}

func TestHandler_Register_DuplicateEmail(t *testing.T) {
	h := NewHandler(NewUseCase(NewInMemoryUserRepo(), nil))
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
	h := NewHandler(NewUseCase(NewInMemoryUserRepo(), nil))
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
