package httpresponse_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/httpresponse"
	"github.com/stretchr/testify/require"
)

func TestWriteJSON(t *testing.T) {
	tests := []struct {
		name       string
		body       any
		wantStatus int
		wantBody   string
	}{
		{name: "success", body: map[string]string{"name": "Anna"}, wantStatus: http.StatusOK, wantBody: `{"name":"Anna"}`},
		{name: "encoding error", body: make(chan int), wantStatus: http.StatusInternalServerError, wantBody: `{"error":"internal server error"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			httpresponse.WriteJSON(rec, http.StatusOK, tt.body, nil)
			require.Equal(t, tt.wantStatus, rec.Code)
			require.JSONEq(t, tt.wantBody, rec.Body.String())
			require.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"))
		})
	}
}

func TestWriteError(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		message string
		want    string
	}{
		{name: "client error", status: http.StatusBadRequest, message: "invalid ID", want: "invalid ID"},
		{name: "internal details hidden", status: http.StatusInternalServerError, message: "postgres password=secret", want: httpresponse.InternalErrorMessage},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			httpresponse.WriteError(rec, tt.status, tt.message, nil)
			var body httpresponse.ErrorResponse
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			require.Equal(t, tt.status, rec.Code)
			require.Equal(t, tt.want, body.Error)
		})
	}
}
