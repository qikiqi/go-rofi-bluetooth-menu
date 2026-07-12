package program

// Device is a bluetooth device reported by bluetoothctl.
type Device struct {
	MAC       string
	Name      string
	Connected bool
}

func symbol(status bool) string {
	if status {
		return "󰂱"
	}
	return "󰂲"
}
