package main

import (
	"os"
	"reflect"
	"testing"

	"github.com/rs/zerolog"
)

func TestMain(m *testing.M) {
	// Silence the global logger; several functions under test (validMAC,
	// writeRofiTempfile) log unconditionally and would otherwise spam
	// test output.
	zerolog.SetGlobalLevel(zerolog.Disabled)
	os.Exit(m.Run())
}

func TestGetSymbol(t *testing.T) {
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
			if got := getSymbol(tt.connected); got != tt.want {
				t.Errorf("getSymbol(%v) = %q, want %q", tt.connected, got, tt.want)
			}
		})
	}
}

func TestSanitizeDevice(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "single device line",
			input: "Device AA:BB:CC:DD:EE:FF My Headphones",
			want:  []string{"AA:BB:CC:DD:EE:FF My Headphones"},
		},
		{
			name: "multiple device lines with controller noise",
			input: "Controller 00:11:22:33:44:55 MyController [default]\n" +
				"Device AA:BB:CC:DD:EE:FF My Headphones\n" +
				"Device 11:22:33:44:55:66 Other Device\n",
			want: []string{
				"AA:BB:CC:DD:EE:FF My Headphones",
				"11:22:33:44:55:66 Other Device",
			},
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
			// "Device" substring is matched even without the trailing
			// space that the subsequent Replace requires, so the line
			// is kept but left un-stripped. Characterizes existing
			// behavior rather than asserting it is desirable.
			name:  "Device substring without trailing space is left unstripped",
			input: "SomeDeviceX line without a space after Device",
			want:  []string{"SomeDeviceX line without a space after Device"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := sanitizeDevice(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("sanitizeDevice(%q) = %#v, want %#v", tt.input, got, tt.want)
			}
		})
	}
}

func TestGetMacFromUserInput(t *testing.T) {
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
			if got := getMacFromUserInput(tt.input); got != tt.want {
				t.Errorf("getMacFromUserInput(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestGetMacFromUserInput_PanicsOnSingleField characterizes a real
// limitation: getMacFromUserInput indexes strings.Fields(input)[1]
// unchecked, so any input with fewer than two whitespace-separated
// fields panics instead of returning an error.
func TestGetMacFromUserInput_PanicsOnSingleField(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for input with fewer than 2 fields, got none")
		}
	}()
	getMacFromUserInput("onlyoneword")
}

func TestGetConnectAction(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "currently connected requests disconnect", input: "󰂱: AA:BB:CC:DD:EE:FF Name", want: "dis"},
		{name: "currently disconnected requests connect", input: "󰂲: AA:BB:CC:DD:EE:FF Name", want: ""},
		{name: "empty input", input: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := getConnectAction(tt.input); got != tt.want {
				t.Errorf("getConnectAction(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidMAC(t *testing.T) {
	devices := map[string]Device{
		"AA:BB:CC:DD:EE:FF My Headphones": {Name: "AA:BB:CC:DD:EE:FF My Headphones", Connected: true},
		"11:22:33:44:55:66 Other Device":  {Name: "11:22:33:44:55:66 Other Device", Connected: false},
	}

	tests := []struct {
		name    string
		input   string
		devices map[string]Device
		want    bool
	}{
		{
			name:    "matching device",
			input:   "󰂱: AA:BB:CC:DD:EE:FF My Headphones",
			devices: devices,
			want:    true,
		},
		{
			name:    "no matching device",
			input:   "󰂲: 99:88:77:66:55:44 Unknown",
			devices: devices,
			want:    false,
		},
		{
			name:    "empty device map",
			input:   "anything",
			devices: map[string]Device{},
			want:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := validMAC(tt.input, tt.devices); got != tt.want {
				t.Errorf("validMAC(%q, devices) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestSortDeviceMapByConnected(t *testing.T) {
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
				"a": {Name: "a", Connected: true},
				"b": {Name: "b", Connected: true},
			},
		},
		{
			name: "all disconnected",
			devices: map[string]Device{
				"a": {Name: "a", Connected: false},
				"b": {Name: "b", Connected: false},
			},
		},
		{
			name: "mixed connected and disconnected",
			devices: map[string]Device{
				"a": {Name: "a", Connected: true},
				"b": {Name: "b", Connected: false},
				"c": {Name: "c", Connected: true},
				"d": {Name: "d", Connected: false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := sortDeviceMapByConnected(tt.devices)

			if len(got) != len(tt.devices) {
				t.Fatalf("sortDeviceMapByConnected() returned %d devices, want %d", len(got), len(tt.devices))
			}

			// Map iteration order is randomized, so only assert the
			// invariant the function promises: every connected device
			// precedes every disconnected device.
			seenDisconnected := false
			for _, d := range got {
				if !d.Connected {
					seenDisconnected = true
				} else if seenDisconnected {
					t.Fatalf("connected device %q found after a disconnected device in %v", d.Name, got)
				}
			}

			gotSet := make(map[string]Device, len(got))
			for _, d := range got {
				gotSet[d.Name] = d
			}
			for name, want := range tt.devices {
				if got, ok := gotSet[name]; !ok || got != want {
					t.Errorf("device %q missing or altered in sorted output: got %+v, want %+v", name, got, want)
				}
			}
		})
	}
}

func TestCreateDeviceMap(t *testing.T) {
	tests := []struct {
		name      string
		connected []string
		paired    []string
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
			connected: []string{"AA:BB My Headphones"},
			paired:    nil,
			want: map[string]Device{
				"AA:BB My Headphones": {Name: "AA:BB My Headphones", Connected: true},
			},
		},
		{
			name:      "paired only",
			connected: nil,
			paired:    []string{"11:22 Other Device"},
			want: map[string]Device{
				"11:22 Other Device": {Name: "11:22 Other Device", Connected: false},
			},
		},
		{
			name:      "connected takes precedence over paired duplicate",
			connected: []string{"AA:BB My Headphones"},
			paired:    []string{"AA:BB My Headphones", "11:22 Other Device"},
			want: map[string]Device{
				"AA:BB My Headphones": {Name: "AA:BB My Headphones", Connected: true},
				"11:22 Other Device":  {Name: "11:22 Other Device", Connected: false},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := createDeviceMap(tt.connected, tt.paired)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("createDeviceMap(%v, %v) = %+v, want %+v", tt.connected, tt.paired, got, tt.want)
			}
		})
	}
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
				{Name: "AA:BB My Headphones", Connected: true},
			},
			want: "󰂱: AA:BB My Headphones\n",
		},
		{
			name: "mixed devices preserve given order",
			devices: []Device{
				{Name: "AA:BB My Headphones", Connected: true},
				{Name: "11:22 Other Device", Connected: false},
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

			writeRofiTempfile(f, tt.devices)

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

// TestUserSelectionDerivesConnectCommand characterizes the full mapping that
// main() relies on: from a rofi selection line to the (MAC, action) pair fed
// into connectDevice. It composes getMacFromUserInput + getConnectAction — the
// two functions the Device-model split will fold into a single by-MAC lookup —
// and locks their combined contract so the split can be proven behavior-
// preserving. The asserted values must survive the refactor unchanged.
func TestUserSelectionDerivesConnectCommand(t *testing.T) {
	tests := []struct {
		name           string
		selection      string
		wantMAC        string
		wantDisconnect string // "dis" => disconnect, "" => connect
	}{
		{
			name:           "connected device selected requests disconnect",
			selection:      "󰂱: AA:BB:CC:DD:EE:FF My Headphones",
			wantMAC:        "AA:BB:CC:DD:EE:FF",
			wantDisconnect: "dis",
		},
		{
			name:           "disconnected device selected requests connect",
			selection:      "󰂲: 11:22:33:44:55:66 Other Device",
			wantMAC:        "11:22:33:44:55:66",
			wantDisconnect: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := getMacFromUserInput(tt.selection); got != tt.wantMAC {
				t.Errorf("getMacFromUserInput(%q) = %q, want %q", tt.selection, got, tt.wantMAC)
			}
			if got := getConnectAction(tt.selection); got != tt.wantDisconnect {
				t.Errorf("getConnectAction(%q) = %q, want %q", tt.selection, got, tt.wantDisconnect)
			}
		})
	}
}
