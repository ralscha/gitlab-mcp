package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireBearerToken(t *testing.T) {
	tests := []struct {
		name, header string
		want         int
	}{
		{"valid", "Bearer secret", http.StatusNoContent},
		{"case insensitive scheme", "bearer secret", http.StatusNoContent},
		{"optional whitespace", "Bearer   secret", http.StatusNoContent},
		{"missing", "", http.StatusUnauthorized},
		{"wrong token", "Bearer nope", http.StatusUnauthorized},
		{"wrong scheme", "Basic secret", http.StatusUnauthorized},
		{"extra value", "Bearer secret extra", http.StatusUnauthorized},
	}
	handler := requireBearerToken("secret", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			req.Header.Set("Authorization", tt.header)
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)
			if recorder.Code != tt.want {
				t.Errorf("status = %d, want %d", recorder.Code, tt.want)
			}
		})
	}
}
