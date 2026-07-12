package version

import (
	"bytes"
	"io"
	"os"
	"runtime"
	"strings"
	"testing"
)

// TestPrint cannot run in parallel with other tests: it swaps out the
// package-global os.Stdout for the duration of the call.
func TestPrint(t *testing.T) {
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = old }()

	if err := Print(); err != nil {
		t.Fatalf("Print() error = %v", err)
	}

	w.Close()
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("io.Copy() error = %v", err)
	}
	out := buf.String()

	// The test binary is built by the Go toolchain, so build info is
	// available and the Go version appears in the output.
	if !strings.Contains(out, runtime.Version()) {
		t.Errorf("Print() output = %q, want it to contain the Go version %q", out, runtime.Version())
	}
}
