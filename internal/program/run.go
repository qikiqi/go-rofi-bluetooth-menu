package program

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/qikiqi/go-rofi-bluetooth-menu/internal/version"
)

// Run executes the rofi Bluetooth menu.
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
		version.PrintVersion()
		os.Exit(0)
	}

	bt := bluetoothctlRunner{}
	menu := rofiMenu{}

	tempFile, err := os.CreateTemp("", "bluetooth")
	if err != nil {
		slog.Error("cannot create tempfile", "err", err)
		os.Exit(1)
	}
	defer os.Remove(tempFile.Name())

	connected := parseDevices(bt.Run(ctx, "devices Connected"))
	paired := parseDevices(bt.Run(ctx, "devices"))

	allDevices := mergeDevices(connected, paired)
	allDevicesSorted := sortByConnected(allDevices)

	writeRofiTempfile(tempFile, allDevicesSorted)

	userInput, err := menu.Prompt(ctx, tempFile.Name())
	if err != nil {
		slog.Error("rofi failed", "err", err, "output", userInput)
		return
	}

	device, err := resolveSelection(userInput, allDevices)
	if err != nil {
		slog.Warn("could not resolve selection", "err", err, "selection", userInput)
		return
	}

	connectDevice(ctx, bt, device)
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
