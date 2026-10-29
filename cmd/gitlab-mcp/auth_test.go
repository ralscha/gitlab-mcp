package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"gitlab-mcp/internal/config"
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

func TestHTTPHandlerCrossOriginProtection(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1"}, nil)
	handler := newHTTPHandler(&config.Config{AuthToken: "secret"}, server)
	const initialize = `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"test","version":"1"}}}`
	for _, tt := range []struct {
		name, origin, fetchSite string
		want                    int
	}{
		{"native client", "", "", http.StatusOK},
		{"same origin", "http://localhost:8080", "", http.StatusOK},
		{"cross origin", "https://untrusted.example", "", http.StatusForbidden},
		{"opaque origin", "null", "", http.StatusForbidden},
		{"cross site fetch", "", "cross-site", http.StatusForbidden},
	} {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "http://localhost:8080/", strings.NewReader(initialize))
			req.Header.Set("Authorization", "Bearer secret")
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept", "application/json, text/event-stream")
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			if tt.fetchSite != "" {
				req.Header.Set("Sec-Fetch-Site", tt.fetchSite)
			}
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			if w.Code != tt.want {
				t.Fatalf("status = %d, want %d; body = %s", w.Code, tt.want, w.Body.String())
			}
		})
	}
}
