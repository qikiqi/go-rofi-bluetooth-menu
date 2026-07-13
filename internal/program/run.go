package program

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/qikiqi/go-rofi-bluetooth-menu/internal/version"
)

// Environment variables rofi sets when executing this binary as a
// script-mode "-modi" mode. See rofi-script(5).
const (
	rofiRetvEnv = "ROFI_RETV"
	rofiInfoEnv = "ROFI_INFO"
)

// rofiNoCustomHeader tells rofi to reject arbitrary typed input and only
// accept one of the listed entries — there is never a device behind
// free-typed text, so ROFI_RETV=2 (custom entry) should be unreachable.
const rofiNoCustomHeader = "\x00no-custom\x1ftrue"

// Run dispatches on whether ROFI_RETV is set. When it is, this process was
// exec'd by rofi as a script-mode mode and Run answers that protocol
// directly (see rofi-script(5)). When it isn't, this is a manual/debug
// invocation: flags are parsed as before and the candidate list is printed
// to stdout for inspection.
func Run(ctx context.Context) {
	if retv, ok := os.LookupEnv(rofiRetvEnv); ok {
		runScriptMode(ctx, retv)
		return
	}

	logLevel := flag.String("loglevel", "info", "set log level: debug, info, warn, error")
	versionFlag := flag.Bool("version", false, "Print the version information")
	vFlag := flag.Bool("v", false, "Print the version information (shorthand)")

	flag.Parse()

	level, err := parseLogLevel(*logLevel)
	if err != nil {
		slog.Error("invalid log level", "err", err)
		os.Exit(1)
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})))

	if *versionFlag || *vFlag {
		if err := version.Print(); err != nil {
			slog.Error("version unavailable", "err", err)
			os.Exit(1)
		}
		return
	}

	exitOnError(listDevices(ctx, bluetoothctlRunner{}, os.Stdout))
}

// runScriptMode answers a rofi script-mode invocation. It never touches the
// flag package: argv[1] on a ROFI_RETV=1 call is a Bluetooth device's
// display name — untrusted-ish, adapter-controlled text — and flag.Parse
// would choke if such a name ever started with "-".
func runScriptMode(ctx context.Context, retv string) {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))

	switch retv {
	case "0":
		exitOnError(listDevices(ctx, bluetoothctlRunner{}, os.Stdout))
	case "1":
		exitOnError(selectDevice(ctx, bluetoothctlRunner{}, os.Getenv(rofiInfoEnv)))
	default:
		slog.Warn("unhandled ROFI_RETV, reprinting list", "retv", retv)
		exitOnError(listDevices(ctx, bluetoothctlRunner{}, os.Stdout))
	}
}

// exitOnError logs a fatal error and exits non-zero, or does nothing if err
// is nil.
func exitOnError(err error) {
	if err != nil {
		slog.Error("fatal error", "err", err)
		os.Exit(1)
	}
}

// listDevices writes one rofi script-mode row per device to w, connected
// devices first, preceded by a header line disabling free-typed input.
func listDevices(ctx context.Context, bt Bluetoothctl, w io.Writer) error {
	devices, err := gatherDevices(ctx, bt)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, rofiNoCustomHeader); err != nil {
		return err
	}
	for _, d := range sortByConnected(devices) {
		if _, err := fmt.Fprintln(w, formatScriptRow(d)); err != nil {
			return err
		}
	}
	return nil
}

// selectDevice resolves mac (from ROFI_INFO) against the current device
// list and toggles it. An unrecognized mac (e.g. the device vanished
// between the list and select calls) is logged and treated as a no-op
// rather than an error.
func selectDevice(ctx context.Context, bt Bluetoothctl, mac string) error {
	devices, err := gatherDevices(ctx, bt)
	if err != nil {
		return err
	}
	device, ok := devices[mac]
	if !ok {
		slog.Warn("could not resolve selection", "mac", mac)
		return nil
	}
	if err := connectDevice(ctx, bt, device); err != nil {
		return fmt.Errorf("toggle %s: %w", device.MAC, err)
	}
	return nil
}

// run lists devices, presents the menu, and toggles the chosen device. It
// returns nil for normal outcomes (including the user dismissing the menu) and
// an error only for genuine failures.
//
// Deprecated: superseded by listDevices/selectDevice via rofi script-mode
// (see Run). Kept only until the menu.go/tempfile path it depends on is
// removed in a follow-up commit.
func run(ctx context.Context, bt Bluetoothctl, menu Menu) error {
	tempFile, err := os.CreateTemp("", "bluetooth")
	if err != nil {
		return fmt.Errorf("create tempfile: %w", err)
	}
	defer func() { _ = tempFile.Close() }()
	defer func() { _ = os.Remove(tempFile.Name()) }()

	allDevices, err := gatherDevices(ctx, bt)
	if err != nil {
		return err
	}
	if err := writeRofiTempfile(tempFile, sortByConnected(allDevices)); err != nil {
		return fmt.Errorf("write menu: %w", err)
	}

	selection, err := menu.Prompt(ctx, tempFile.Name())
	if err != nil {
		if errors.Is(err, ErrNoSelection) {
			slog.Info("no selection made")
			return nil
		}
		return fmt.Errorf("prompt: %w", err)
	}

	device, err := resolveSelection(selection, allDevices)
	if err != nil {
		slog.Warn("could not resolve selection", "err", err, "selection", selection)
		return nil
	}

	if err := connectDevice(ctx, bt, device); err != nil {
		return fmt.Errorf("toggle %s: %w", device.MAC, err)
	}
	return nil
}

// parseLogLevel maps a -loglevel string to a slog.Level.
func parseLogLevel(s string) (slog.Level, error) {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("unknown log level %q", s)
	}
}
