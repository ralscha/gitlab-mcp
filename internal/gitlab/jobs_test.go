package gitlab

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestGetJob(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.EscapedPath() != "/api/v4/projects/group%2Fproject/jobs/42" {
			t.Errorf("request = %s %s", r.Method, r.URL.EscapedPath())
		}
		_, _ = io.WriteString(w, `{"id":42,"name":"test","stage":"test","status":"failed","failure_reason":"script_failure","pipeline":{"id":8}}`)
	}))
	t.Cleanup(server.Close)
	client, err := NewClient(server.URL, "token", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	job, err := client.GetJob(t.Context(), "group/project", 42)
	if err != nil {
		t.Fatal(err)
	}
	if job.ID != 42 || job.Status != "failed" || job.FailureReason != "script_failure" || job.Pipeline == nil || job.Pipeline.ID != 8 {
		t.Fatalf("job = %+v", job)
	}
}

func TestGetJobTrace(t *testing.T) {
	for _, tt := range []struct {
		name   string
		status int
		body   string
		large  bool
	}{
		{name: "text", status: http.StatusOK, body: "Running tests\nFAIL: example\n"},
		{name: "empty", status: http.StatusOK},
		{name: "at limit", status: http.StatusOK, body: strings.Repeat("x", MaxJobTraceBytes)},
		{name: "over limit", status: http.StatusOK, body: strings.Repeat("x", MaxJobTraceBytes+1), large: true},
		{name: "missing", status: http.StatusNotFound, body: `{"message":"404 Job Not Found"}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.EscapedPath() != "/gitlab/api/v4/projects/group%2Fproject/jobs/42/trace" {
					t.Errorf("request = %s %s", r.Method, r.URL.EscapedPath())
				}
				if r.Header.Get("PRIVATE-TOKEN") != "token" || r.Header.Get("Accept") != "text/plain" {
					t.Error("missing authentication or text response header")
				}
				w.WriteHeader(tt.status)
				_, _ = io.WriteString(w, tt.body)
			}))
			t.Cleanup(server.Close)
			client, err := NewClient(server.URL+"/gitlab", "token", server.Client())
			if err != nil {
				t.Fatal(err)
			}
			trace, err := client.GetJobTrace(t.Context(), "group/project", 42)
			switch {
			case tt.large:
				if !errors.Is(err, ErrTooLarge) || trace != "" {
					t.Fatalf("oversized trace: len=%d, err=%v", len(trace), err)
				}
			case tt.status == http.StatusNotFound:
				var apiErr *APIError
				if !errors.As(err, &apiErr) || apiErr.StatusCode != tt.status || apiErr.Message != "404 Job Not Found" {
					t.Fatalf("error = %v", err)
				}
			default:
				if err != nil || trace != tt.body {
					t.Fatalf("trace mismatch: len=%d, err=%v", len(trace), err)
				}
			}
		})
	}
}

func TestGetJobTraceRetriesRateLimit(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if attempts.Add(1) == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = io.WriteString(w, "passed\n")
	}))
	t.Cleanup(server.Close)
	client, err := NewClient(server.URL, "token", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	trace, err := client.GetJobTrace(t.Context(), "1", 42)
	if err != nil || trace != "passed\n" || attempts.Load() != 2 {
		t.Fatalf("trace=%q, attempts=%d, err=%v", trace, attempts.Load(), err)
	}
}
