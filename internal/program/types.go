package program

type Device struct {
	Name      string
	Connected bool
}

func symbol(status bool) string {
	if status {
		return "󰂱"
	}
	return "󰂲"
}
