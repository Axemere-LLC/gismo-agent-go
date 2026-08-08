package agent

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBearerAuth(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := BearerAuth("secret-key", inner)

	tests := []struct {
		name       string
		authHeader string
		wantStatus int
	}{
		{"correct key", "Bearer secret-key", http.StatusOK},
		{"wrong key", "Bearer wrong-key", http.StatusUnauthorized},
		{"missing header", "", http.StatusUnauthorized},
		{"missing Bearer prefix", "secret-key", http.StatusUnauthorized},
		{"key is a prefix of another header value", "Bearer secret-key-extra", http.StatusUnauthorized},
		{"empty bearer value", "Bearer ", http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestBearerAuth_EmptyKeyRejectsEverything(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := BearerAuth("", inner)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Authorization", "Bearer ")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d (an empty configured key must never match)", rec.Code, http.StatusUnauthorized)
	}
}
