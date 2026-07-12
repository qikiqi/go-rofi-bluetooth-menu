package program

import (
	"context"
	"os/exec"
)

// Menu presents the pre-rendered device list and returns the user's selection.
type Menu interface {
	Prompt(ctx context.Context, tempFileName string) (string, error)
}

var _ Menu = rofiMenu{}

// rofiMenu is the Menu backed by the real rofi binary in -dmenu mode.
type rofiMenu struct{}

func (rofiMenu) Prompt(ctx context.Context, tempFileName string) (string, error) {
	cmd := exec.Command("rofi", "-dmenu", "-input", tempFileName, "-i", "-p", "Bluetooth", "-keep-right")
	output, err := cmd.Output()
	return string(output), err
}
