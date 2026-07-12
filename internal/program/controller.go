package program

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// bluetoothctlTimeout bounds a single bluetoothctl invocation so a stalled
// adapter cannot hang the menu indefinitely. Generous enough to cover a real
// device connection, which can take several seconds.
const bluetoothctlTimeout = 30 * time.Second

// Bluetoothctl runs a command against the system bluetoothctl and returns its
// output.
type Bluetoothctl interface {
	Run(ctx context.Context, command string) (string, error)
}

var _ Bluetoothctl = bluetoothctlRunner{}

// bluetoothctlRunner is the Bluetoothctl backed by the real bluetoothctl binary.
// The command is fed on stdin, exactly as an interactive session would receive
// it; bluetoothctl runs it and exits on EOF.
type bluetoothctlRunner struct{}

func (bluetoothctlRunner) Run(ctx context.Context, command string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, bluetoothctlTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bluetoothctl")
	cmd.Stdin = strings.NewReader(command + "\n")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("bluetoothctl %q: %w", command, err)
	}
	return string(out), nil
}

// connectDevice powers the adapter on and toggles the device: a currently
// connected device is disconnected, otherwise it is connected.
func connectDevice(ctx context.Context, bt Bluetoothctl, device Device) error {
	if _, err := bt.Run(ctx, "power on"); err != nil {
		return err
	}
	action := "connect"
	if device.Connected {
		action = "disconnect"
	}
	if _, err := bt.Run(ctx, action+" "+device.MAC); err != nil {
		return err
	}
	return nil
}
