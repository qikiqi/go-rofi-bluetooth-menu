package program

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/qikiqi/go-rofi-bluetooth-menu/internal/version"
)

// Run parses flags, configures logging, and executes the rofi Bluetooth menu,
// exiting non-zero on a fatal error.
func Run(ctx context.Context) {
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

	if err := run(ctx, bluetoothctlRunner{}, rofiMenu{}); err != nil {
		slog.Error("fatal error", "err", err)
		os.Exit(1)
	}
}

// run lists devices, presents the menu, and toggles the chosen device. It
// returns nil for normal outcomes (including the user dismissing the menu) and
// an error only for genuine failures.
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
