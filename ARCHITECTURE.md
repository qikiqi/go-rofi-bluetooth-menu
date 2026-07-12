# Architecture

`go-rofi-bluetooth-menu` is a single-shot CLI: it lists Bluetooth devices,
renders a rofi menu, and toggles the chosen device. It runs once and exits — it
is not a daemon.

## Package layout

```
main.go                      # signal.NotifyContext → program.Run(ctx)
internal/
├── program/
│   ├── run.go               # Run(ctx): flags, slog setup; run(ctx, bt, menu) orchestration
│   ├── types.go             # Device{MAC, Name, Connected}; symbol()
│   ├── controller.go        # Bluetoothctl interface + bluetoothctlRunner impl
│   ├── menu.go              # Menu interface + rofiMenu impl; ErrNoSelection
│   └── devices.go           # pure logic: parseDevices, mergeDevices, sortByConnected,
│                            #   writeRofiTempfile, selectedMAC, resolveSelection
└── version/
    └── version.go           # Print(): build/version info from debug.ReadBuildInfo
```

`main` stays intentionally tiny: it installs a signal-cancelled context and
hands off to `program.Run`. All logic lives under `internal/`, so nothing is
part of an importable public API.

## Dependency seams

The two external integrations are expressed as small consumer-side interfaces,
defined where they are used (`program`) rather than where they are implemented.
Both have a single production implementation and a compile-time conformance
check.

```go
type Bluetoothctl interface {
    Run(ctx context.Context, command string) (string, error)
}

type Menu interface {
    Prompt(ctx context.Context, tempFileName string) (string, error)
}
```

`run(ctx, bt Bluetoothctl, menu Menu)` accepts these interfaces, so tests inject
a `fakeBluetoothctl` and exercise the full flow without touching real hardware
or a display server. The real implementations shell out with
`exec.CommandContext`.

## Data flow

```
bluetoothctl "devices Connected" ─┐
                                  ├─ parseDevices ─ mergeDevices ─ sortByConnected ─ writeRofiTempfile ─→ rofi
bluetoothctl "devices" ───────────┘                                                                        │
                                                                                                           ▼
                                    connectDevice ←── resolveSelection ←──────────────────────────── user picks a line
```

- **parseDevices** turns each `Device <MAC> <name...>` line into a `Device`,
  skipping anything that isn't a device line.
- **mergeDevices** unions the connected and paired lists into a MAC-keyed map,
  marking connected devices.
- **writeRofiTempfile** renders `<symbol>: <MAC> <name>` lines to a temp file
  that rofi reads via `-input`.
- **resolveSelection** maps the picked line back to its `Device` by the MAC
  embedded in it (`selectedMAC`), returning an error instead of panicking on a
  malformed selection.
- **connectDevice** powers the adapter on and issues `connect` or `disconnect`
  based on the device's current state.

## Execution and error model

- Each `bluetoothctl` invocation is bounded by a 30s context timeout. `rofi` is
  cancellable but has no deadline, since the user may take arbitrarily long to
  choose.
- `run` returns an error only for genuine failures (temp file, `bluetoothctl`,
  `rofi` launch); `Run` logs it and exits `1`. Normal outcomes — including the
  user dismissing the menu (`ErrNoSelection`) or picking an unknown entry —
  return `nil` and exit `0`.
- All logging goes through `log/slog` to stderr; verbosity is set by
  `-loglevel`.

## Testing

- Pure logic (`parseDevices`, `mergeDevices`, `sortByConnected`,
  `writeRofiTempfile`, `selectedMAC`, `resolveSelection`) is covered by
  table-driven unit tests.
- `TestSelectionIssuesConnectCommand` is a characterization test that pins the
  exact `bluetoothctl` commands a given menu selection produces.
- `controller.go` / `menu.go` implementations are tested against hermetic stub
  scripts placed on `PATH`, so no real `bluetoothctl`/`rofi` is needed.
