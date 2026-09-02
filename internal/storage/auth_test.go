package storage

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireStorageToken(t *testing.T) {
	const token = "test-storage-token"

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	handler := RequireStorageToken(token, next)

	tests := []struct {
		name          string
		authorization string
		wantStatus    int
	}{
		{
			name:       "missing token",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:          "wrong token",
			authorization: "Bearer wrong-token",
			wantStatus:    http.StatusUnauthorized,
		},
		{
			name:          "valid token",
			authorization: "Bearer " + token,
			wantStatus:    http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			if tt.authorization != "" {
				req.Header.Set("Authorization", tt.authorization)
			}
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, req)

			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, tt.wantStatus)
			}
		})
	}
}
