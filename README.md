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

Run it directly, or bind it to a key in your window manager. For Sway:

```
bindsym $mod+b exec go-rofi-bluetooth-menu
```

The menu shows one entry per device as `<symbol>: <MAC> <name>`, connected
devices first:

```
󰂱: AA:BB:CC:DD:EE:FF My Headphones
󰂲: 11:22:33:44:55:66 Living Room Speaker
```

Picking a **connected** device (󰂱) disconnects it; picking a **disconnected**
device (󰂲) powers the adapter on and connects it. Dismissing the menu without
choosing anything does nothing.

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-loglevel` | `info` | Log verbosity: `debug`, `info`, `warn`, `error` |
| `-version`, `-v` | | Print version and build info, then exit |

Logs are written to stderr as structured `log/slog` text.

### Exit codes

- `0` — success, including when the menu is dismissed with no selection.
- `1` — a fatal error (invalid `-loglevel`, `bluetoothctl` failure, `rofi`
  could not be launched, etc.). The reason is logged before exit.

`SIGINT` / `SIGTERM` cancel any in-flight subprocess and exit cleanly.

## Development

```sh
go build ./...
go vet ./...
go test ./...
```

The Bluetooth and rofi integrations sit behind interfaces, so the tests run
against fakes and hermetic stub scripts — no real hardware or display server
required. See [ARCHITECTURE.md](ARCHITECTURE.md) for the package layout.

## License

MIT — see [LICENSE](LICENSE).
