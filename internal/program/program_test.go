package program

import (
	"context"
	"errors"
	"log/slog"
	"maps"
	"os"
	"slices"
	"testing"
)

func TestMain(m *testing.M) {
	// Silence structured logging; several functions under test log
	// unconditionally and would otherwise spam test output.
	slog.SetDefault(slog.New(slog.DiscardHandler))
	os.Exit(m.Run())
}

// fakeBluetoothctl records the commands it is asked to run. outputs, if set,
// overrides output on a per-command basis; err, if set, is returned for every
// call instead of output.
type fakeBluetoothctl struct {
	commands []string
	output   string
	outputs  map[string]string
	err      error
}

func (f *fakeBluetoothctl) Run(_ context.Context, command string) (string, error) {
	f.commands = append(f.commands, command)
	if f.err != nil {
		return "", f.err
	}

	if out, ok := f.outputs[command]; ok {
		return out, nil
	}

	return f.output, nil
}

func TestSymbol(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		connected bool
		want      string
	}{
		{name: "connected", connected: true, want: "󰂱"},
		{name: "disconnected", connected: false, want: "󰂲"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := symbol(tt.connected); got != tt.want {
				t.Errorf("symbol(%v) = %q, want %q", tt.connected, got, tt.want)
			}
		})
	}
}

func TestParseDevices(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  []Device
	}{
		{
			name:  "single device line",
			input: "Device AA:BB:CC:DD:EE:FF My Headphones",
			want:  []Device{{MAC: "AA:BB:CC:DD:EE:FF", Name: "My Headphones"}},
		},
		{
			name: "multiple device lines with controller noise",
			input: "Controller 00:11:22:33:44:55 MyController [default]\n" +
				"Device AA:BB:CC:DD:EE:FF My Headphones\n" +
				"Device 11:22:33:44:55:66 Other Device\n",
			want: []Device{
				{MAC: "AA:BB:CC:DD:EE:FF", Name: "My Headphones"},
				{MAC: "11:22:33:44:55:66", Name: "Other Device"},
			},
		},
		{
			name:  "device without a name",
			input: "Device AA:BB:CC:DD:EE:FF",
			want:  []Device{{MAC: "AA:BB:CC:DD:EE:FF", Name: ""}},
		},
		{
			name:  "no matching lines",
			input: "Controller 00:11:22:33:44:55 MyController [default]\n",
			want:  nil,
		},
		{
			name:  "empty input",
			input: "",
			want:  nil,
		},
		{
			// Only lines that actually start with "Device " are devices; a
			// stray "Device" substring elsewhere is not a device line. This
			// corrects the old Contains-based parse, which kept such lines.
			name:  "Device substring midline is not a device",
			input: "SomeDeviceX line without a space after Device",
			want:  nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := parseDevices(tt.input)
			if !slices.Equal(got, tt.want) {
				t.Errorf("parseDevices(%q) = %#v, want %#v", tt.input, got, tt.want)
			}
		})
	}
}

func TestSortByConnected(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		devices map[string]Device
	}{
		{
			name:    "empty map",
			devices: map[string]Device{},
		},
		{
			name: "all connected",
			devices: map[string]Device{
				"a": {MAC: "a", Connected: true},
				"b": {MAC: "b", Connected: true},
			},
		},
		{
			name: "all disconnected",
			devices: map[string]Device{
				"a": {MAC: "a", Connected: false},
				"b": {MAC: "b", Connected: false},
			},
		},
		{
			name: "mixed connected and disconnected",
			devices: map[string]Device{
				"a": {MAC: "a", Connected: true},
				"b": {MAC: "b", Connected: false},
				"c": {MAC: "c", Connected: true},
				"d": {MAC: "d", Connected: false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := sortByConnected(tt.devices)

			if len(got) != len(tt.devices) {
				t.Fatalf("sortByConnected() returned %d devices, want %d", len(got), len(tt.devices))
			}

			// Map iteration order is randomized, so only assert the
			// invariant the function promises: every connected device
			// precedes every disconnected device.
			seenDisconnected := false
			for _, d := range got {
				if !d.Connected {
					seenDisconnected = true
				} else if seenDisconnected {
					t.Fatalf("connected device %q found after a disconnected device in %v", d.MAC, got)
				}
			}

			gotSet := make(map[string]Device, len(got))
			for _, d := range got {
				gotSet[d.MAC] = d
			}

			for mac, want := range tt.devices {
				if got, ok := gotSet[mac]; !ok || got != want {
					t.Errorf("device %q missing or altered in sorted output: got %+v, want %+v", mac, got, want)
				}
			}
		})
	}
}

func TestMergeDevices(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		connected []Device
		paired    []Device
		want      map[string]Device
	}{
		{
			name:      "empty inputs",
			connected: nil,
			paired:    nil,
			want:      map[string]Device{},
		},
		{
			name:      "connected only",
			connected: []Device{{MAC: "AA:BB", Name: "My Headphones"}},
			paired:    nil,
			want: map[string]Device{
				"AA:BB": {MAC: "AA:BB", Name: "My Headphones", Connected: true},
			},
		},
		{
			name:      "paired only",
			connected: nil,
			paired:    []Device{{MAC: "11:22", Name: "Other Device"}},
			want: map[string]Device{
				"11:22": {MAC: "11:22", Name: "Other Device", Connected: false},
			},
		},
		{
			name:      "connected takes precedence over paired duplicate",
			connected: []Device{{MAC: "AA:BB", Name: "My Headphones"}},
			paired: []Device{
				{MAC: "AA:BB", Name: "My Headphones"},
				{MAC: "11:22", Name: "Other Device"},
			},
			want: map[string]Device{
				"AA:BB": {MAC: "AA:BB", Name: "My Headphones", Connected: true},
				"11:22": {MAC: "11:22", Name: "Other Device", Connected: false},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := mergeDevices(tt.connected, tt.paired)
			if !maps.Equal(got, tt.want) {
				t.Errorf("mergeDevices(%v, %v) = %+v, want %+v", tt.connected, tt.paired, got, tt.want)
			}
		})
	}
}

func TestGatherDevices(t *testing.T) {
	t.Run("merges connected and paired output", func(t *testing.T) {
		t.Parallel()

		bt := &fakeBluetoothctl{outputs: map[string]string{
			"devices Connected": "Device AA:BB:CC:DD:EE:FF My Headphones\n",
			"devices":           "Device AA:BB:CC:DD:EE:FF My Headphones\nDevice 11:22:33:44:55:66 Other Device\n",
		}}

		got, err := gatherDevices(t.Context(), bt)
		if err != nil {
			t.Fatalf("gatherDevices() error = %v", err)
		}

		want := map[string]Device{
			"AA:BB:CC:DD:EE:FF": {MAC: "AA:BB:CC:DD:EE:FF", Name: "My Headphones", Connected: true},
			"11:22:33:44:55:66": {MAC: "11:22:33:44:55:66", Name: "Other Device", Connected: false},
		}
		if !maps.Equal(got, want) {
			t.Errorf("gatherDevices() = %+v, want %+v", got, want)
		}

		if want := []string{"devices Connected", "devices"}; !slices.Equal(bt.commands, want) {
			t.Errorf("gatherDevices() issued commands %v, want %v", bt.commands, want)
		}
	})

	t.Run("propagates bluetoothctl error", func(t *testing.T) {
		t.Parallel()

		bt := &fakeBluetoothctl{err: errors.New("boom")}

		if _, err := gatherDevices(t.Context(), bt); err == nil {
			t.Error("gatherDevices() error = nil, want an error when bluetoothctl fails")
		}
	})
}

func TestFormatScriptRow(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		device Device
		want   string
	}{
		{
			name:   "connected device with name",
			device: Device{MAC: "AA:BB:CC:DD:EE:FF", Name: "My Headphones", Connected: true},
			want:   "󰂱: AA:BB:CC:DD:EE:FF My Headphones\x00info\x1fAA:BB:CC:DD:EE:FF",
		},
		{
			name:   "disconnected device with name",
			device: Device{MAC: "11:22:33:44:55:66", Name: "Other Device", Connected: false},
			want:   "󰂲: 11:22:33:44:55:66 Other Device\x00info\x1f11:22:33:44:55:66",
		},
		{
			name:   "device without a name has no trailing space",
			device: Device{MAC: "AA:BB", Connected: true},
			want:   "󰂱: AA:BB\x00info\x1fAA:BB",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := formatScriptRow(tt.device); got != tt.want {
				t.Errorf("formatScriptRow(%+v) = %q, want %q", tt.device, got, tt.want)
			}
		})
	}
}
