package program

type Device struct {
	Name      string
	Connected bool
}

func getSymbol(status bool) string {
	if status {
		return "󰂱"
	}
	return "󰂲"
}
