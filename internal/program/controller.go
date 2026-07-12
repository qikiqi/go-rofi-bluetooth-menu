package program

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"

	"github.com/rs/zerolog/log"
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
	err := cmd.Run()
	if err != nil {
		log.Error().Msgf("Error running bluetoothctl: %v", err)
	}
	return out.String()
}

func connectDevice(ctx context.Context, bt Bluetoothctl, mac string, disconnect string) {
	bt.Run(ctx, "power on")
	log.Error().Msgf("%sconnect %s", disconnect, mac)
	bt.Run(ctx, fmt.Sprintf("%sconnect %s", disconnect, mac))
}
