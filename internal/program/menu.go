package program

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// ErrNoSelection is returned by Menu.Prompt when the user dismissed the menu
// without choosing an entry.
var ErrNoSelection = errors.New("no selection made")

// Menu presents the pre-rendered device list and returns the user's selection.
type Menu interface {
	Prompt(ctx context.Context, tempFileName string) (string, error)
}

var _ Menu = rofiMenu{}

// rofiMenu is the Menu backed by the real rofi binary in -dmenu mode. It honors
// ctx cancellation (e.g. a signal) but sets no deadline: the user may take an
// arbitrary amount of time to pick an entry.
type rofiMenu struct{}

func (rofiMenu) Prompt(ctx context.Context, tempFileName string) (string, error) {
	cmd := exec.CommandContext(ctx, "rofi", "-dmenu", "-input", tempFileName, "-i", "-p", "Bluetooth", "-keep-right")
	output, err := cmd.Output()
	if err != nil {
		// rofi exits non-zero when the user dismisses the menu without
		// picking anything; treat that as a normal no-selection outcome
		// rather than a failure.
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return "", ErrNoSelection
		}
		return "", fmt.Errorf("run rofi: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}
