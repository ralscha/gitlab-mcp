package main

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestHTTPTransportListTools(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()
	addr := "127.0.0.1:" + strconv.Itoa(port)
	exe := filepath.Join(t.TempDir(), "gitlab-mcp.exe")
	//nolint:gosec // The executable output is an isolated test temp path.
	build := exec.CommandContext(ctx, "go", "build", "-o", exe, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, output)
	}
	//nolint:gosec // exe is the binary built above in an isolated test temp path.
	cmd := exec.CommandContext(ctx, exe, "--gitlab-token=test-token", "--mode=readwrite", "--transport=http", "--http-addr="+addr, "--auth-token=mcp-secret")
	cmd.Env = os.Environ()
	var logs lockedBuffer
	cmd.Stdout, cmd.Stderr = &logs, &logs
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	exitCh := make(chan error, 1)
	go func() { exitCh <- cmd.Wait() }()
	defer func() {
		if cmd.ProcessState == nil {
			_ = cmd.Process.Kill()
			<-exitCh
		}
	}()

	transport := &mcp.StreamableClientTransport{
		Endpoint:   "http://" + addr,
		HTTPClient: &http.Client{Transport: bearerRoundTripper{token: "mcp-secret"}},
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "smoke-test", Version: "0"}, nil)
	var session *mcp.ClientSession
	deadline := time.Now().Add(20 * time.Second)
	for {
		session, err = client.Connect(ctx, transport, nil)
		if err == nil {
			break
		}
		select {
		case processErr := <-exitCh:
			t.Fatalf("server exited: %v\n%s", processErr, logs.String())
		default:
		}
		if time.Now().After(deadline) {
			t.Fatalf("Connect() error = %v\n%s", err, logs.String())
		}
		time.Sleep(200 * time.Millisecond)
	}
	defer func() { _ = session.Close() }()
	result, err := session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		t.Fatal(err)
	}
	foundRead, foundWrite := false, false
	for _, tool := range result.Tools {
		if tool.Name == "gitlab_get_project" {
			foundRead = true
		}
		if tool.Name == "gitlab_create_issue" {
			foundWrite = true
		}
	}
	if !foundRead || !foundWrite {
		t.Errorf("read=%v write=%v", foundRead, foundWrite)
	}
}

type bearerRoundTripper struct{ token string }

func (c bearerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.Header.Set("Authorization", "Bearer "+c.token)
	return http.DefaultTransport.RoundTrip(clone)
}

type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}
func (b *lockedBuffer) String() string { b.mu.Lock(); defer b.mu.Unlock(); return b.buf.String() }
