package httprequest_test

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/httprequest"
)

func TestDecodeJSON(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{name: "object", body: `{"name":"Anna"}`},
		{name: "trailing whitespace", body: "{\"name\":\"Anna\"} \n"},
		{name: "empty", wantErr: true},
		{name: "null", body: `null`, wantErr: true},
		{name: "unknown field", body: `{"role":"admin"}`, wantErr: true},
		{name: "two objects", body: `{} {}`, wantErr: true},
		{name: "trailing garbage", body: `{} !`, wantErr: true},
		{name: "array", body: `[]`, wantErr: true},
		{name: "wrong type", body: `{"name":1}`, wantErr: true},
		{name: "oversized body", body: `{"name":"` + strings.Repeat("a", httprequest.MaxBodyBytes) + `"}`, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var value struct {
				Name string `json:"name"`
			}
			r := httptest.NewRequest("POST", "/", strings.NewReader(tt.body))
			err := httprequest.DecodeJSON(httptest.NewRecorder(), r, &value)
			if tt.wantErr {
				if !errors.Is(err, httprequest.ErrInvalidBody) {
					t.Fatalf("want invalid body, got %v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}
