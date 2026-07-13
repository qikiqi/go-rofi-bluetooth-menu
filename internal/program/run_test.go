package program

import (
	"bytes"
	"errors"
	"slices"
	"strings"
	"testing"
)

func TestListDevices(t *testing.T) {
	t.Run("writes header and script rows for each device", func(t *testing.T) {
		t.Parallel()
		bt := &fakeBluetoothctl{outputs: map[string]string{
			"devices Connected": "Device AA:BB:CC:DD:EE:FF My Headphones\n",
			"devices":           "Device AA:BB:CC:DD:EE:FF My Headphones\nDevice 11:22:33:44:55:66 Other Device\n",
		}}

		var buf bytes.Buffer
		if err := listDevices(t.Context(), bt, &buf); err != nil {
			t.Fatalf("listDevices() error = %v", err)
		}

		got := buf.String()
		if !strings.HasPrefix(got, rofiNoCustomHeader+"\n") {
			t.Errorf("listDevices() output %q, want it to start with the no-custom header", got)
		}
		for _, want := range []string{
			"AA:BB:CC:DD:EE:FF My Headphones\x00info\x1fAA:BB:CC:DD:EE:FF",
			"11:22:33:44:55:66 Other Device\x00info\x1f11:22:33:44:55:66",
		} {
			if !strings.Contains(got, want) {
				t.Errorf("listDevices() output %q, want it to contain %q", got, want)
			}
		}
	})

	t.Run("propagates bluetoothctl error", func(t *testing.T) {
		t.Parallel()
		bt := &fakeBluetoothctl{err: errors.New("boom")}
		var buf bytes.Buffer
		if err := listDevices(t.Context(), bt, &buf); err == nil {
			t.Error("listDevices() error = nil, want an error when bluetoothctl fails")
		}
	})
}

func TestSelectDevice(t *testing.T) {
	t.Run("toggles the device matching mac", func(t *testing.T) {
		t.Parallel()
		bt := &fakeBluetoothctl{outputs: map[string]string{
			"devices Connected": "Device AA:BB:CC:DD:EE:FF My Headphones\n",
			"devices":           "Device AA:BB:CC:DD:EE:FF My Headphones\n",
		}}

		if err := selectDevice(t.Context(), bt, "AA:BB:CC:DD:EE:FF"); err != nil {
			t.Fatalf("selectDevice() error = %v", err)
		}

		want := []string{"devices Connected", "devices", "power on", "disconnect AA:BB:CC:DD:EE:FF"}
		if !slices.Equal(bt.commands, want) {
			t.Errorf("selectDevice() issued commands %v, want %v", bt.commands, want)
		}
	})

	t.Run("unknown mac is a no-op, not an error", func(t *testing.T) {
		t.Parallel()
		bt := &fakeBluetoothctl{}

		if err := selectDevice(t.Context(), bt, "99:88:77:66:55:44"); err != nil {
			t.Fatalf("selectDevice() error = %v, want nil for an unrecognized mac", err)
		}
		for _, cmd := range bt.commands {
			if strings.HasPrefix(cmd, "connect") || strings.HasPrefix(cmd, "disconnect") {
				t.Errorf("selectDevice() issued %q for an unrecognized mac, want no toggle", cmd)
			}
		}
	})

	t.Run("propagates bluetoothctl error", func(t *testing.T) {
		t.Parallel()
		bt := &fakeBluetoothctl{err: errors.New("boom")}
		if err := selectDevice(t.Context(), bt, "AA:BB"); err == nil {
			t.Error("selectDevice() error = nil, want an error when bluetoothctl fails")
		}
	})
}
