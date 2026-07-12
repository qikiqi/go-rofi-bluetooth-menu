package program

import (
	"bytes"
	"fmt"
	"os/exec"

	"github.com/rs/zerolog/log"
)

func runBluetoothctl(command string) string {
	cmd := exec.Command("bash", "-c", fmt.Sprintf("echo -e \"%s\" | bluetoothctl", command))
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		log.Error().Msgf("Error running bluetoothctl: %v", err)
	}
	return out.String()
}

func connectDevice(mac string, disconnect string) {
	runBluetoothctl("power on")
	log.Error().Msgf("%sconnect %s", disconnect, mac)
	runBluetoothctl(fmt.Sprintf("%sconnect %s", disconnect, mac))
}
