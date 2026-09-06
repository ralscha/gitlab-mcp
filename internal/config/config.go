// Package config loads and validates gitlab-mcp configuration.
package config

import (
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"
)

var ErrVersionRequested = errors.New("version requested")

type Mode string

const (
	ModeReadOnly  Mode = "readonly"
	ModeReadWrite Mode = "readwrite"
)

type Transport string

const (
	TransportStdio Transport = "stdio"
	TransportHTTP  Transport = "http"
)

type Config struct {
	GitLabBaseURL string
	GitLabToken   string
	Mode          Mode
	Transport     Transport
	HTTPAddr      string
	AuthToken     string
}

// Load reads environment defaults and applies command-line flag overrides.
// GITLAB_BASE_URL defaults to GitLab.com. GITLAB_TOKEN is always required.
func Load(args []string) (*Config, error) {
	cfg := &Config{
		GitLabBaseURL: "https://gitlab.com",
		GitLabToken:   os.Getenv("GITLAB_TOKEN"),
		Mode:          ModeReadOnly,
		Transport:     TransportStdio,
		HTTPAddr:      ":8080",
		AuthToken:     os.Getenv("MCP_AUTH_TOKEN"),
	}
	if value := os.Getenv("GITLAB_BASE_URL"); value != "" {
		cfg.GitLabBaseURL = value
	}
	if value := os.Getenv("GITLAB_MODE"); value != "" {
		cfg.Mode = Mode(value)
	}
	if value := os.Getenv("MCP_TRANSPORT"); value != "" {
		cfg.Transport = Transport(value)
	}
	if value := os.Getenv("MCP_HTTP_ADDR"); value != "" {
		cfg.HTTPAddr = value
	}

	fs := flag.NewFlagSet("gitlab-mcp", flag.ContinueOnError)
	baseURL := fs.String("gitlab-base-url", cfg.GitLabBaseURL, "GitLab instance URL, e.g. https://gitlab.com")
	token := fs.String("gitlab-token", cfg.GitLabToken, "GitLab personal, project, or group access token")
	mode := fs.String("mode", string(cfg.Mode), "Server mode: readonly or readwrite")
	transport := fs.String("transport", string(cfg.Transport), "Transport: stdio or http")
	httpAddr := fs.String("http-addr", cfg.HTTPAddr, "Address to listen on when --transport=http")
	authToken := fs.String("auth-token", cfg.AuthToken, "Bearer token required from HTTP MCP clients")
	showVersion := fs.Bool("version", false, "Print the gitlab-mcp version and exit")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if *showVersion {
		return nil, ErrVersionRequested
	}

	cfg.GitLabBaseURL = strings.TrimRight(*baseURL, "/")
	cfg.GitLabToken = *token
	cfg.Mode = Mode(*mode)
	cfg.Transport = Transport(*transport)
	cfg.HTTPAddr = *httpAddr
	cfg.AuthToken = *authToken
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	var problems []string
	parsed, err := url.Parse(c.GitLabBaseURL)
	switch {
	case err != nil:
		problems = append(problems, fmt.Sprintf("GITLAB_BASE_URL is not a valid URL: %v", err))
	case parsed.Scheme != "https" || parsed.Host == "":
		problems = append(problems, "GITLAB_BASE_URL must be an absolute https URL")
	case parsed.User != nil:
		problems = append(problems, "GITLAB_BASE_URL must not contain user information")
	case parsed.RawQuery != "" || parsed.Fragment != "":
		problems = append(problems, "GITLAB_BASE_URL must not contain a query string or fragment")
	}
	if c.GitLabToken == "" {
		problems = append(problems, "GITLAB_TOKEN (or --gitlab-token) is required")
	}
	switch c.Mode {
	case ModeReadOnly, ModeReadWrite:
	default:
		problems = append(problems, fmt.Sprintf("invalid mode %q: must be %q or %q", c.Mode, ModeReadOnly, ModeReadWrite))
	}
	switch c.Transport {
	case TransportStdio, TransportHTTP:
	default:
		problems = append(problems, fmt.Sprintf("invalid transport %q: must be %q or %q", c.Transport, TransportStdio, TransportHTTP))
	}
	if len(problems) > 0 {
		return fmt.Errorf("invalid configuration:\n  %s", strings.Join(problems, "\n  "))
	}
	return nil
}

func (c *Config) IsReadWrite() bool { return c.Mode == ModeReadWrite }
