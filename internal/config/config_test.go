package config

import (
	"errors"
	"strings"
	"testing"
)

func clearEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{"GITLAB_BASE_URL", "GITLAB_TOKEN", "GITLAB_MODE", "MCP_TRANSPORT", "MCP_HTTP_ADDR", "MCP_AUTH_TOKEN"} {
		t.Setenv(key, "")
	}
}

func TestLoadDefaults(t *testing.T) {
	clearEnv(t)
	t.Setenv("GITLAB_TOKEN", "secret")
	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.GitLabBaseURL != "https://gitlab.com" {
		t.Errorf("GitLabBaseURL = %q", cfg.GitLabBaseURL)
	}
	if cfg.Mode != ModeReadOnly {
		t.Errorf("Mode = %q", cfg.Mode)
	}
	if cfg.Transport != TransportStdio {
		t.Errorf("Transport = %q", cfg.Transport)
	}
	if cfg.HTTPAddr != ":8080" {
		t.Errorf("HTTPAddr = %q", cfg.HTTPAddr)
	}
	if cfg.IsReadWrite() {
		t.Error("IsReadWrite() = true")
	}
}

func TestLoadFlagsOverrideEnvironment(t *testing.T) {
	clearEnv(t)
	t.Setenv("GITLAB_BASE_URL", "https://env.example.com")
	t.Setenv("GITLAB_TOKEN", "env-token")
	t.Setenv("GITLAB_MODE", "readonly")
	t.Setenv("MCP_TRANSPORT", "stdio")
	cfg, err := Load([]string{
		"--gitlab-base-url=https://flag.example.com/", "--gitlab-token=flag-token",
		"--mode=readwrite", "--transport=http", "--http-addr=127.0.0.1:9090", "--auth-token=mcp-token",
	})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.GitLabBaseURL != "https://flag.example.com" {
		t.Errorf("GitLabBaseURL = %q", cfg.GitLabBaseURL)
	}
	if cfg.GitLabToken != "flag-token" {
		t.Errorf("GitLabToken = %q", cfg.GitLabToken)
	}
	if !cfg.IsReadWrite() || cfg.Transport != TransportHTTP || cfg.HTTPAddr != "127.0.0.1:9090" || cfg.AuthToken != "mcp-token" {
		t.Errorf("unexpected config: %#v", cfg)
	}
}

func TestLoadValidation(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"missing token", nil, "GITLAB_TOKEN"},
		{"http URL", []string{"--gitlab-base-url=http://gitlab.example.com", "--gitlab-token=x"}, "absolute https"},
		{"query URL", []string{"--gitlab-base-url=https://gitlab.example.com?x=1", "--gitlab-token=x"}, "query string"},
		{"userinfo URL", []string{"--gitlab-base-url=https://user:password@gitlab.example.com", "--gitlab-token=x"}, "user information"},
		{"bad mode", []string{"--gitlab-token=x", "--mode=write"}, "invalid mode"},
		{"bad transport", []string{"--gitlab-token=x", "--transport=sse"}, "invalid transport"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv(t)
			_, err := Load(tt.args)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Load() error = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func TestVersionShortCircuitsValidation(t *testing.T) {
	clearEnv(t)
	_, err := Load([]string{"--version"})
	if !errors.Is(err, ErrVersionRequested) {
		t.Fatalf("Load() error = %v", err)
	}
}
