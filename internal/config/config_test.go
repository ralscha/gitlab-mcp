package config

import (
	"bytes"
	"errors"
	"flag"
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
		{"positional argument", []string{"--gitlab-token=x", "unexpected"}, "positional arguments"},
		{"flags after positional argument", []string{"unexpected", "--gitlab-token=x"}, "positional arguments"},
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

func TestUsageDoesNotExposeCredentials(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"--unknown-flag"}, {"--gitlab-token=flag-gitlab-secret", "--auth-token=flag-mcp-secret", "--help"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			clearEnv(t)
			t.Setenv("GITLAB_TOKEN", "env-gitlab-secret")
			t.Setenv("MCP_AUTH_TOKEN", "env-mcp-secret")
			var output bytes.Buffer
			_, err := load(args, &output)
			if err == nil {
				t.Fatal("expected help or flag parse error")
			}
			if args[len(args)-1] == "--help" && !errors.Is(err, flag.ErrHelp) {
				t.Fatalf("error = %v, want flag.ErrHelp", err)
			}
			if !strings.Contains(output.String(), "Usage of gitlab-mcp") {
				t.Fatalf("missing usage output: %s", output.String())
			}
			for _, secret := range []string{"env-gitlab-secret", "env-mcp-secret", "flag-gitlab-secret", "flag-mcp-secret"} {
				if strings.Contains(output.String(), secret) {
					t.Error("usage output exposed a credential")
				}
			}
		})
	}
}

func TestCredentialDefaultsAndEmptyOverrides(t *testing.T) {
	clearEnv(t)
	t.Setenv("GITLAB_TOKEN", "env-gitlab-secret")
	t.Setenv("MCP_AUTH_TOKEN", "env-mcp-secret")
	cfg, err := Load(nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.GitLabToken != "env-gitlab-secret" || cfg.AuthToken != "env-mcp-secret" {
		t.Fatal("environment credentials were not preserved")
	}
	cfg, err = Load([]string{"--auth-token="})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AuthToken != "" {
		t.Error("explicit empty auth token did not override environment")
	}
	_, err = Load([]string{"--gitlab-token="})
	if err == nil || !strings.Contains(err.Error(), "GITLAB_TOKEN") {
		t.Fatalf("explicit empty GitLab token: error = %v", err)
	}
}

func TestVersionShortCircuitsValidation(t *testing.T) {
	clearEnv(t)
	_, err := Load([]string{"--version"})
	if !errors.Is(err, ErrVersionRequested) {
		t.Fatalf("Load() error = %v", err)
	}
}
