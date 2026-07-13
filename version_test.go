package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

// TestPrintVersion cannot run in parallel with other tests: it swaps out
// the package-global os.Stdout for the duration of the call.
func TestPrintVersion(t *testing.T) {
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = old }()

	PrintVersion()

	w.Close()
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("io.Copy() error = %v", err)
	}
	out := buf.String()

	// debug.ReadBuildInfo() succeeds for binaries built by the Go
	// toolchain, which includes the `go test` binary itself, so the
	// "no build info" branch isn't exercised here.
	for _, want := range []string{"Build Info:", "Go Version:"} {
		if !strings.Contains(out, want) {
			t.Errorf("PrintVersion() output = %q, want it to contain %q", out, want)
		}
	}
}
