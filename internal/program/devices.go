package program

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strings"
)

// parseDevices extracts devices from `bluetoothctl devices`-style output. Each
// device line has the form "Device <MAC> <name...>"; lines that do not start
// with "Device " or that carry no MAC field are skipped. Connected is left
// false — mergeDevices assigns it.
func parseDevices(input string) []Device {
	var devices []Device
	scanner := bufio.NewScanner(strings.NewReader(input))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "Device ") {
			continue
		}
		mac, name, ok := splitMACName(strings.TrimPrefix(line, "Device "))
		if !ok {
			continue
		}
		devices = append(devices, Device{MAC: mac, Name: name})
	}
	return devices
}

// gatherDevices lists connected and paired devices via bt and merges them
// into a MAC-keyed map, marking which are currently connected.
func gatherDevices(ctx context.Context, bt Bluetoothctl) (map[string]Device, error) {
	connectedOut, err := bt.Run(ctx, "devices Connected")
	if err != nil {
		return nil, fmt.Errorf("list connected devices: %w", err)
	}
	pairedOut, err := bt.Run(ctx, "devices")
	if err != nil {
		return nil, fmt.Errorf("list paired devices: %w", err)
	}
	return mergeDevices(parseDevices(connectedOut), parseDevices(pairedOut)), nil
}

// splitMACName splits "<MAC> <name...>" into its MAC and (possibly empty) name.
// ok is false when no MAC is present.
func splitMACName(s string) (mac, name string, ok bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", "", false
	}
	parts := strings.SplitN(s, " ", 2)
	mac = parts[0]
	if len(parts) == 2 {
		name = strings.TrimSpace(parts[1])
	}
	return mac, name, true
}

// mergeDevices combines connected and paired devices into a MAC-keyed map,
// marking devices from the connected list as Connected. A device present in
// both keeps its connected status.
func mergeDevices(connected, paired []Device) map[string]Device {
	all := make(map[string]Device)
	for _, d := range connected {
		d.Connected = true
		all[d.MAC] = d
	}
	for _, d := range paired {
		if _, exists := all[d.MAC]; !exists {
			d.Connected = false
			all[d.MAC] = d
		}
	}
	return all
}

func sortByConnected(devices map[string]Device) []Device {
	list := make([]Device, 0, len(devices))
	for _, d := range devices {
		list = append(list, d)
	}
	slices.SortFunc(list, func(a, b Device) int {
		switch {
		case a.Connected == b.Connected:
			return 0
		case a.Connected:
			return -1
		default:
			return 1
		}
	})
	return list
}

// writeRofiTempfile renders one menu line per device as "<symbol>: <MAC> <name>"
// (the name is omitted, with no trailing space, when empty).
func writeRofiTempfile(tempFile *os.File, devices []Device) error {
	for _, d := range devices {
		entry := strings.TrimSpace(d.MAC + " " + d.Name)
		slog.Debug("menu entry", "symbol", symbol(d.Connected), "mac", d.MAC, "name", d.Name)
		if _, err := fmt.Fprintf(tempFile, "%s: %s\n", symbol(d.Connected), entry); err != nil {
			return err
		}
	}
	return nil
}

// formatScriptRow renders a device as a rofi script-mode row: the same
// "<symbol>: <MAC> <name>" text writeRofiTempfile produces, plus a hidden
// \0info\x1f<MAC> field so the follow-up ROFI_RETV=1 call gets the MAC via
// the ROFI_INFO env var instead of it having to be parsed back out of the
// display text.
func formatScriptRow(d Device) string {
	entry := strings.TrimSpace(d.MAC + " " + d.Name)
	return fmt.Sprintf("%s: %s\x00info\x1f%s", symbol(d.Connected), entry, d.MAC)
}

// selectedMAC extracts the MAC from a rendered menu selection of the form
// "<symbol>: <MAC> <name>", returning an error instead of panicking when the
// selection has no MAC field.
func selectedMAC(selection string) (string, error) {
	fields := strings.Fields(selection)
	if len(fields) < 2 {
		return "", fmt.Errorf("selection %q has no MAC field", selection)
	}
	return fields[1], nil
}

// resolveSelection maps a menu selection back to the Device it refers to,
// looking it up by the MAC embedded in the selection line.
func resolveSelection(selection string, devices map[string]Device) (Device, error) {
	mac, err := selectedMAC(selection)
	if err != nil {
		return Device{}, err
	}
	device, ok := devices[mac]
	if !ok {
		return Device{}, fmt.Errorf("selection %q (mac %q) not among known devices", selection, mac)
	}
	return device, nil
}
