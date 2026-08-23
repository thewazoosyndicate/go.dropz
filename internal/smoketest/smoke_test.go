//go:build smoke

// End-to-end smoke test: boots the real binary and makes one gRPC call.
// Guards the wiring no unit test touches (flag parsing, protobuf descriptor
// loading, server startup, JSON log contract, graceful shutdown).
// A corrupted generated descriptor once panicked at startup with all unit
// tests green; this test exists so that never ships again.
//
// Run: go build -o bin/dropz cmd/dropz/main.go && DROPZ_BIN=bin/dropz go test -tags smoke ./internal/smoketest
package smoketest

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/dropz/dropz/internal/protocol"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found above %s", dir)
		}
		dir = parent
	}
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func TestBinarySmoke(t *testing.T) {
	bin := os.Getenv("DROPZ_BIN")
	if bin == "" {
		t.Skip("DROPZ_BIN not set")
	}
	// Tests run in the package dir; resolve relative paths from the repo root.
	abs := bin
	if !filepath.IsAbs(abs) {
		root, err := repoRoot()
		if err != nil {
			t.Fatal(err)
		}
		abs = filepath.Join(root, bin)
	}
	if _, err := os.Stat(abs); err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	addr := fmt.Sprintf("127.0.0.1:%d", freePort(t))

	cmd := exec.Command(abs,
		"--data-dir", filepath.Join(dir, "data"),
		"--video-dir", filepath.Join(dir, "videos"),
		"--log-dir", filepath.Join(dir, "logs"),
		"--log-format", "json",
		"--server-addr", addr,
	)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer cmd.Process.Kill()

	// Every stderr line must be valid JSON: that is the Electron host contract.
	// Violations are collected and asserted after Wait; t.Errorf from this
	// goroutine could fire after the test ends and panic the test binary.
	ready := make(chan struct{})
	lines := make(chan string, 256)
	var mu sync.Mutex
	var badLines []string
	scanDone := make(chan struct{})
	go func() {
		defer close(scanDone)
		scanner := bufio.NewScanner(stderr)
		readySeen := false
		for scanner.Scan() {
			line := scanner.Text()
			select {
			case lines <- line:
			default:
			}
			var entry struct {
				Msg string `json:"msg"`
			}
			if err := json.Unmarshal([]byte(line), &entry); err != nil {
				mu.Lock()
				badLines = append(badLines, line)
				mu.Unlock()
				continue
			}
			if entry.Msg == "gRPC server started" && !readySeen {
				readySeen = true
				close(ready)
			}
		}
	}()

	select {
	case <-ready:
	case <-time.After(20 * time.Second):
		t.Fatalf("server never became ready; logs:\n%s", drain(lines))
	}

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := protocol.NewDropzServiceClient(conn).GetConfig(ctx, &protocol.GetConfigRequest{})
	if err != nil {
		t.Fatalf("GetConfig: %v; logs:\n%s", err, drain(lines))
	}
	if resp.GetConfig() == nil || resp.GetConfig().GetScanIntervalSeconds() <= 0 {
		t.Fatalf("GetConfig returned implausible config: %+v", resp.GetConfig())
	}

	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("binary exited uncleanly: %v; logs:\n%s", err, drain(lines))
		}
	case <-time.After(20 * time.Second):
		t.Fatalf("binary did not exit on SIGTERM; logs:\n%s", drain(lines))
	}

	<-scanDone
	mu.Lock()
	defer mu.Unlock()
	for _, l := range badLines {
		t.Errorf("non-JSON log line: %q", l)
	}
}

func drain(lines chan string) string {
	var out string
	for {
		select {
		case l := <-lines:
			out += l + "\n"
		default:
			return out
		}
	}
}
