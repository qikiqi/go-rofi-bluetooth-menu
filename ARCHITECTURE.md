# Architecture

`go-rofi-bluetooth-menu` is a rofi
[script-mode](https://github.com/davatorium/rofi/blob/next/doc/rofi-script.5.markdown)
`-modi`: rofi execs this binary, not the other way around. It runs once per
call and exits — it is not a daemon, and (unlike an earlier version of this
tool) it never spawns rofi itself.

## Package layout

```
main.go                      # signal.NotifyContext → program.Run(ctx)
internal/
├── program/
│   ├── run.go               # Run(ctx): ROFI_RETV dispatch, listDevices, selectDevice
│   ├── types.go             # Device{MAC, Name, Connected}; symbol()
│   ├── controller.go        # Bluetoothctl interface + bluetoothctlRunner impl
│   └── devices.go           # pure logic: parseDevices, mergeDevices, sortByConnected,
│                            #   gatherDevices, formatScriptRow
└── version/
    └── version.go           # Print(): build/version info from debug.ReadBuildInfo
```

`main` stays intentionally tiny: it installs a signal-cancelled context and
hands off to `program.Run`. All logic lives under `internal/`, so nothing is
part of an importable public API.

## The rofi script-mode protocol

Rofi calls the executable directly as a `-modi`, setting the `ROFI_RETV`
environment variable to say why:

- `ROFI_RETV` unset or `0` — initial call. Print one line per device to
  stdout; that becomes the menu.
- `ROFI_RETV=1` — the user picked a line. `ROFI_INFO` carries whatever
  hidden payload that row was tagged with.
- Anything else (`2` custom text, `3` deleted entry, `10+` custom
  keybindings) — not opted into; `Run` logs it and reprints the list rather
  than doing nothing silently.

Each printed row can carry a payload invisible to the matcher via a
`\0info\x1f<value>` suffix. This binary tags every row with its MAC, so the
`ROFI_RETV=1` call reads the MAC straight out of `ROFI_INFO` — no parsing
the picked line's text back into a MAC. The very first output line is
`\0no-custom\x1ftrue`, a mode-level directive telling rofi to reject
free-typed input entirely, since there's never a device behind arbitrary
text.

## Dependency seam

The one external integration is a small consumer-side interface, defined
where it's used (`program`) rather than where it's implemented, with a
single production implementation and a compile-time conformance check:

```go
type Bluetoothctl interface {
    Run(ctx context.Context, command string) (string, error)
}
```

`listDevices`/`selectDevice` accept this interface, so tests inject a
`fakeBluetoothctl` and exercise the real logic without touching real
hardware. The real implementation shells out with `exec.CommandContext`.

## Data flow

```
                 ROFI_RETV=0 (or unset)
bluetoothctl "devices Connected" ─┐
                                  ├─ gatherDevices ─ sortByConnected ─ formatScriptRow ─→ stdout ─→ rofi renders
bluetoothctl "devices" ───────────┘

                 ROFI_RETV=1, ROFI_INFO=<mac>
bluetoothctl "devices Connected" ─┐
                                  ├─ gatherDevices ─ lookup by mac ─→ connectDevice
bluetoothctl "devices" ───────────┘
```

- **gatherDevices** issues both `bluetoothctl` listings and merges them into
  a MAC-keyed map — shared by both the list and select paths, so a select
  call always toggles against live state rather than a snapshot from
  whenever the list was drawn.
- **formatScriptRow** renders `<symbol>: <MAC> <name>` plus the hidden
  `\0info\x1f<MAC>` field.
- **selectDevice** looks the picked MAC up directly in a fresh
  `gatherDevices` map; an unrecognized MAC (e.g. the device vanished between
  calls) is logged and treated as a no-op, not an error.
- **connectDevice** powers the adapter on and issues `connect` or
  `disconnect` based on the device's current state.

## Execution and error model

- Each `bluetoothctl` invocation is bounded by a 30s context timeout.
- `Run` checks `ROFI_RETV` *before* touching the `flag` package at all.
  On a real `ROFI_RETV=1` call, `argv[1]` is a Bluetooth device's display
  name — adapter/peer-controlled text `flag.Parse` has no business seeing.
  The flag surface (`-loglevel`, `-version`/`-v`) only exists on the
  genuinely-manual, no-`ROFI_RETV` invocation path.
- `listDevices`/`selectDevice` return an error only for genuine failures
  (a `bluetoothctl` call failing); `Run`/`runScriptMode` log it via
  `exitOnError` and exit `1`. An unresolved selection returns `nil` and
  exits `0`, same as a normal toggle.
- All logging goes through `log/slog` to stderr. Verbosity is `-loglevel` on
  the manual path; script-mode invocations (exec'd by rofi, not a terminal)
  use a fixed level.

## Testing

- Pure logic (`parseDevices`, `mergeDevices`, `sortByConnected`,
  `gatherDevices`, `formatScriptRow`) is covered by table-driven unit tests;
  `formatScriptRow`'s tests assert the exact wire bytes rofi parses.
- `listDevices`/`selectDevice` are tested directly against a
  `fakeBluetoothctl`, including the same connect/disconnect
  characterization the old menu-selection flow used to pin.
- `controller.go`'s `bluetoothctlRunner` is tested against a hermetic stub
  script on `PATH`, so no real `bluetoothctl` is needed. There's no
  equivalent rofi-stub test anymore — Go never invokes rofi, so there's
  nothing to stub; the actual rofi round-trip is verified manually
  (`rofi -show bluetooth -modi "bluetooth:<path>"`), not by the test suite.
