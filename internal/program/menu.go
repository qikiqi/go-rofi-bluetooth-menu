package program

import "os/exec"

func runRofi(tempFileName string) (string, error) {
	cmd := exec.Command("rofi", "-dmenu", "-input", tempFileName, "-i", "-p", "Bluetooth", "-keep-right")
	output, err := cmd.Output()
	return string(output), err
}
