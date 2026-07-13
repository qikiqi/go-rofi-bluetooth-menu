package program

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// These tests shell out to real subprocesses, but every external binary the
// program invokes (bluetoothctl, rofi) is a hermetic stub script written to a
// temp directory and prepended to PATH, so none of them touch real Bluetooth
// hardware or a display server. That keeps them fast and deterministic enough
// to run as regular unit tests rather than behind an integration build tag.
//
// None of these tests can use t.Parallel(): t.Setenv panics if called from
// a parallel test.

// stubExecutable writes an executable POSIX shell script named `name`
// into a fresh temp directory, prepends that directory to PATH for the
// duration of the test, and returns the directory.
func stubExecutable(t *testing.T, name, scriptBody string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	content := "#!/bin/sh\n" + scriptBody + "\n"
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", path, err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return dir
}

func TestRunBluetoothctl(t *testing.T) {
	stubExecutable(t, "bluetoothctl", "cat")

	got, err := bluetoothctlRunner{}.Run(t.Context(), "devices")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !strings.Contains(got, "devices") {
		t.Errorf("Run(%q) = %q, want output to contain the piped command", "devices", got)
	}
}

func TestRunBluetoothctl_MissingBinary(t *testing.T) {
	// PATH points at an empty directory, so bluetoothctl cannot be found;
	// Run logs the error and returns empty output.
	t.Setenv("PATH", t.TempDir())

	got, err := bluetoothctlRunner{}.Run(t.Context(), "devices")
	if err == nil {
		t.Error("Run() error = nil, want an error when bluetoothctl is missing")
	}
	if got != "" {
		t.Errorf("Run() = %q, want empty output when bluetoothctl is missing", got)
	}
}

func TestRunRofi(t *testing.T) {
	dir := t.TempDir()
	argsFile := filepath.Join(dir, "args.txt")
	script := fmt.Sprintf(`echo "$@" > %q
echo "selected line"`, argsFile)
	if err := os.WriteFile(filepath.Join(dir, "rofi"), []byte("#!/bin/sh\n"+script+"\n"), 0o755); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	tempFileName := filepath.Join(dir, "menu-input")
	got, err := rofiMenu{}.Prompt(t.Context(), tempFileName)
	if err != nil {
		t.Fatalf("Prompt() error = %v", err)
	}
	if got != "selected line" {
		t.Errorf("Prompt() = %q, want %q", got, "selected line")
	}

	args, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("os.ReadFile(args) error = %v", err)
	}
	if !strings.Contains(string(args), tempFileName) {
		t.Errorf("rofi invoked with args %q, want them to contain the tempfile name %q", args, tempFileName)
	}
}

func TestConnectDevice(t *testing.T) {
	dir := t.TempDir()
	logFile := filepath.Join(dir, "invocations.log")
	stubExecutable(t, "bluetoothctl", fmt.Sprintf("cat >> %q", logFile))

	if err := connectDevice(t.Context(), bluetoothctlRunner{}, Device{MAC: "AA:BB:CC:DD:EE:FF", Connected: true}); err != nil {
		t.Fatalf("connectDevice() error = %v", err)
	}

	got, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}

	for _, want := range []string{"power on", "disconnect AA:BB:CC:DD:EE:FF"} {
		if !strings.Contains(string(got), want) {
			t.Errorf("bluetoothctl invocation log = %q, want it to contain %q", got, want)
		}
	}
}
