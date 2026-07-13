# go-rofi-bluetooth-menu

A small [rofi](https://github.com/davatorium/rofi) menu for connecting and
disconnecting Bluetooth devices through `bluetoothctl`. It lists your paired
devices, marks the connected ones, and toggles whichever you pick.

## Requirements

- `bluetoothctl` (BlueZ) on `PATH`
- `rofi` on `PATH`
- A [Nerd Font](https://www.nerdfonts.com/) for the Bluetooth glyphs (󰂱 / 󰂲)

## Install

```sh
go install github.com/qikiqi/go-rofi-bluetooth-menu@latest
```

Or build from a checkout:

```sh
go build -o go-rofi-bluetooth-menu .
```

## Usage

This is a rofi [script-mode](https://github.com/davatorium/rofi/blob/next/doc/rofi-script.5.markdown)
`-modi`: rofi execs the binary itself, so it's invoked through `rofi -modi`,
not run directly. Bind it to a key in your window manager. For Sway:

```
bindsym $mod+b exec rofi -show bluetooth -modi "bluetooth:/path/to/go-rofi-bluetooth-menu"
```

The menu shows one entry per device as `<symbol>: <MAC> <name>`, connected
devices first:

```
󰂱: AA:BB:CC:DD:EE:FF My Headphones
󰂲: 11:22:33:44:55:66 Living Room Speaker
```

Picking a **connected** device (󰂱) disconnects it; picking a **disconnected**
device (󰂲) powers the adapter on and connects it. Dismissing the menu without
choosing anything does nothing. Free-typed text that doesn't match a device
is rejected by rofi itself — there's no action to take on it.

### Manual invocation

Running the binary directly (no `ROFI_RETV` in the environment, i.e. outside
of rofi) prints the same rows rofi would receive, to stdout, for
inspection — it does not open a menu:

| Flag | Default | Description |
|------|---------|-------------|
| `-loglevel` | `info` | Log verbosity: `debug`, `info`, `warn`, `error` |
| `-version`, `-v` | | Print version and build info, then exit |

Logs are written to stderr as structured `log/slog` text.

### Exit codes

- `0` — success, including when a picked line no longer matches a known
  device.
- `1` — a fatal error (invalid `-loglevel`, a `bluetoothctl` call failing,
  etc.). The reason is logged before exit.

`SIGINT` / `SIGTERM` cancel any in-flight subprocess and exit cleanly.

## Development

```sh
go build ./...
go vet ./...
go test ./...
```

The `bluetoothctl` integration sits behind an interface, so the tests run
against a fake and a hermetic stub script — no real hardware required. See
[ARCHITECTURE.md](ARCHITECTURE.md) for the package layout and the
script-mode protocol details.

## License

MIT — see [LICENSE](LICENSE).
