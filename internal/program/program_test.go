package program

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"maps"
	"os"
	"slices"
	"testing"
)

func TestMain(m *testing.M) {
	// Silence structured logging; several functions under test log
	// unconditionally and would otherwise spam test output.
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
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

func TestSelectedMAC(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "connected symbol prefix",
			input: "󰂱: AA:BB:CC:DD:EE:FF My Headphones",
			want:  "AA:BB:CC:DD:EE:FF",
		},
		{
			name:  "disconnected symbol prefix",
			input: "󰂲: 11:22:33:44:55:66 Other Device",
			want:  "11:22:33:44:55:66",
		},
		{
			name:  "extra leading and inter-field whitespace",
			input: "   󰂱:   AA:BB:CC:DD:EE:FF Name",
			want:  "AA:BB:CC:DD:EE:FF",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := selectedMAC(tt.input)
			if err != nil {
				t.Fatalf("selectedMAC(%q) error = %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("selectedMAC(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestSelectedMAC_ErrorsOnSingleField pins the deliberate fix for the old
// strings.Fields(input)[1] panic: a selection with fewer than two fields now
// returns an error instead of crashing.
func TestSelectedMAC_ErrorsOnSingleField(t *testing.T) {
	t.Parallel()
	if _, err := selectedMAC("onlyoneword"); err == nil {
		t.Fatal("expected error for input with fewer than 2 fields, got nil")
	}
}

func TestResolveSelection(t *testing.T) {
	devices := map[string]Device{
		"AA:BB:CC:DD:EE:FF": {MAC: "AA:BB:CC:DD:EE:FF", Name: "My Headphones", Connected: true},
		"11:22:33:44:55:66": {MAC: "11:22:33:44:55:66", Name: "Other Device"},
	}

	t.Run("resolves known device by MAC", func(t *testing.T) {
		t.Parallel()
		got, err := resolveSelection("󰂱: AA:BB:CC:DD:EE:FF My Headphones", devices)
		if err != nil {
			t.Fatalf("resolveSelection() error = %v", err)
		}
		if want := devices["AA:BB:CC:DD:EE:FF"]; got != want {
			t.Errorf("resolveSelection() = %+v, want %+v", got, want)
		}
	})

	errorCases := []struct {
		name      string
		selection string
	}{
		{name: "no MAC field", selection: "onlyoneword"},
		{name: "unknown MAC", selection: "󰂲: 99:88:77:66:55:44 Unknown"},
		{name: "empty selection", selection: ""},
	}
	for _, tt := range errorCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := resolveSelection(tt.selection, devices); err == nil {
				t.Errorf("resolveSelection(%q) = nil error, want error", tt.selection)
			}
		})
	}
}

// TestSelectionIssuesConnectCommand is the end-to-end characterization the
// Device-model split had to preserve: a given menu selection must still produce
// exactly the same bluetoothctl commands as before the split.
func TestSelectionIssuesConnectCommand(t *testing.T) {
	devices := map[string]Device{
		"AA:BB:CC:DD:EE:FF": {MAC: "AA:BB:CC:DD:EE:FF", Name: "My Headphones", Connected: true},
		"11:22:33:44:55:66": {MAC: "11:22:33:44:55:66", Name: "Other Device", Connected: false},
	}
	tests := []struct {
		name      string
		selection string
		wantCmds  []string
	}{
		{
			name:      "connected device selected requests disconnect",
			selection: "󰂱: AA:BB:CC:DD:EE:FF My Headphones",
			wantCmds:  []string{"power on", "disconnect AA:BB:CC:DD:EE:FF"},
		},
		{
			name:      "disconnected device selected requests connect",
			selection: "󰂲: 11:22:33:44:55:66 Other Device",
			wantCmds:  []string{"power on", "connect 11:22:33:44:55:66"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			device, err := resolveSelection(tt.selection, devices)
			if err != nil {
				t.Fatalf("resolveSelection(%q) error = %v", tt.selection, err)
			}
			bt := &fakeBluetoothctl{}
			if err := connectDevice(t.Context(), bt, device); err != nil {
				t.Fatalf("connectDevice() error = %v", err)
			}
			if !slices.Equal(bt.commands, tt.wantCmds) {
				t.Errorf("connectDevice issued %v, want %v", bt.commands, tt.wantCmds)
			}
		})
	}
}

func TestSortByConnected(t *testing.T) {
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

func TestWriteRofiTempfile(t *testing.T) {
	tests := []struct {
		name    string
		devices []Device
		want    string
	}{
		{
			name:    "empty devices",
			devices: nil,
			want:    "",
		},
		{
			name: "single connected device",
			devices: []Device{
				{MAC: "AA:BB", Name: "My Headphones", Connected: true},
			},
			want: "󰂱: AA:BB My Headphones\n",
		},
		{
			name: "device without a name has no trailing space",
			devices: []Device{
				{MAC: "AA:BB", Name: "", Connected: true},
			},
			want: "󰂱: AA:BB\n",
		},
		{
			name: "mixed devices preserve given order",
			devices: []Device{
				{MAC: "AA:BB", Name: "My Headphones", Connected: true},
				{MAC: "11:22", Name: "Other Device", Connected: false},
			},
			want: "󰂱: AA:BB My Headphones\n󰂲: 11:22 Other Device\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f, err := os.CreateTemp(t.TempDir(), "rofi-tempfile-*")
			if err != nil {
				t.Fatalf("os.CreateTemp() error = %v", err)
			}
			defer f.Close()

			if err := writeRofiTempfile(f, tt.devices); err != nil {
				t.Fatalf("writeRofiTempfile() error = %v", err)
			}

			got, err := os.ReadFile(f.Name())
			if err != nil {
				t.Fatalf("os.ReadFile() error = %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("writeRofiTempfile() wrote %q, want %q", got, tt.want)
			}
		})
	}
}
