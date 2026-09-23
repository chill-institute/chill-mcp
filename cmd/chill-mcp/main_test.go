package main

import (
	"bytes"
	"context"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunUsageAndVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run(nil, &stdout, &stderr); code != 2 || !strings.Contains(stderr.String(), "usage") {
		t.Fatalf("run(nil) = %d, stderr %q", code, stderr.String())
	}
	stderr.Reset()
	if code := run([]string{"bogus"}, &stdout, &stderr); code != 2 {
		t.Fatalf("run(bogus) = %d", code)
	}
	if code := run([]string{"version"}, &stdout, &stderr); code != 0 || !strings.HasPrefix(stdout.String(), "chill-mcp dev") {
		t.Fatalf("run(version) = %d, stdout %q", code, stdout.String())
	}
}

func TestStdioHelpWritesToStderr(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"stdio", "--help"}, &stdout, &stderr); code != 2 || !strings.Contains(stderr.String(), "-profile") {
		t.Fatalf("run(stdio --help) = %d, stderr %q", code, stderr.String())
	}
}

func TestStdioRequiresToken(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{"api_base_url":"https://api.example.test"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := run([]string{"stdio", "--config", path}, &stdout, &stderr); code != 1 || !strings.Contains(stderr.String(), "chilly auth login") {
		t.Fatalf("run(stdio) = %d, stderr %q", code, stderr.String())
	}
}

func TestHTTPServesHealthAndStops(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	_, port, _ := net.SplitHostPort(listener.Addr().String())
	_ = listener.Close()
	t.Setenv(envListenHost, "127.0.0.1")
	t.Setenv(envListenPort, port)
	t.Setenv(envAPIBaseURL, "http://127.0.0.1:9")

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- runHTTP(ctx, slog.New(slog.DiscardHandler)) }()

	deadline := time.Now().Add(5 * time.Second)
	for {
		if err := runHealth(context.Background()); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("health never became ready")
		}
		time.Sleep(20 * time.Millisecond)
	}
	response, err := http.Post("http://127.0.0.1:"+port+"/mcp", "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status without bearer = %d", response.StatusCode)
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("runHTTP() error = %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("server did not stop")
	}
	if err := runHealth(context.Background()); err == nil {
		t.Fatal("health still succeeds after shutdown")
	}
}
