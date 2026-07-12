package program

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os/exec"
)

// Bluetoothctl runs a command against the system bluetoothctl and returns its
// output.
type Bluetoothctl interface {
	Run(ctx context.Context, command string) string
}

var _ Bluetoothctl = bluetoothctlRunner{}

// bluetoothctlRunner is the Bluetoothctl backed by the real bluetoothctl binary.
type bluetoothctlRunner struct{}

func (bluetoothctlRunner) Run(ctx context.Context, command string) string {
	cmd := exec.Command("bash", "-c", fmt.Sprintf("echo -e \"%s\" | bluetoothctl", command))
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		slog.Error("bluetoothctl failed", "command", command, "err", err)
	}
	return out.String()
}

func connectDevice(ctx context.Context, bt Bluetoothctl, mac string, disconnect string) {
	bt.Run(ctx, "power on")
	slog.Debug("bluetoothctl action", "cmd", fmt.Sprintf("%sconnect %s", disconnect, mac))
	bt.Run(ctx, fmt.Sprintf("%sconnect %s", disconnect, mac))
}
