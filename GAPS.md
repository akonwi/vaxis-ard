# Binding gaps

vaxis Go API surface not yet exposed in the Ard bindings. Grouped by the
reason they are not implemented.

## Deferred — blocked

### `EventContext.Notify` — terminal notification support

Bound as `ui::notify(ctx, title, body)`. The underlying vaxis Go function
uses **OSC 777** (`\x1b]777;notify;title;body\x1b\\`) when a non‑empty
title is passed — this is the foot terminal protocol. Passing an empty
title falls back to **OSC 9** (`\x1b]9;body\x1b\\`). Neither sequence
is widely supported:

| Terminal | OSC 9 | OSC 777 |
|----------|-------|--------|
| iTerm2   | ✓     | ✗      |
| Ghostty  | ✗     | ✗      |
| foot     | ?     | ✓      |
| WezTerm  | ✓     | ?      |
| Kitty    | ✗     | ✗      |

No further action on our side — this is a terminal capability issue, not
a binding gap. Documented here for visibility.

### `encode_cells` / `StyledString.encode()`

Serializes `[Cell]` into an ANSI SGR‑encoded string. The implementation is
written and passes as a standalone Ard file, but the Ard compiler panics
(`TypeVar.get` → `$unknown`) when the same code is placed in a library
consumed via `[dependencies]`. The root cause is a compiler bug (the
checker does not recover gracefully from a type‑resolution failure inside a
`match` expression with name‑capture patterns on a type union).
Tracked at: https://github.com/akonwi/ard/issues/222

### `ParseStyledString`

Parses an SGR‑annotated string into `[Cell]`. vaxis Go implements this with
its own `ansi` sub‑package. Binding or porting that parser is a substantial
effort. Blocked on having a working `encode_cells` for round‑trip testing,
and on deciding whether to write a pure‑Ard ANSI parser or bind the Go one.

---

## Deferred — heavy effort

### `widgets/term.Terminal` — embedded terminal emulator widget

Upstream's demo has a Terminal page that spawns `$SHELL -i` inside a
PTY and renders an interactive terminal inside a 72×8 cell area of the
vaxis UI. Useful capability but a non-trivial bind:

- New extern type for `*exec.Cmd` (or a wrapper) plus a constructor
  `terminal_command(name: Str, args: [Str]) TerminalCommand`. `$SHELL`
  lookup is available via stdlib `ard/env` so the demo wiring stays
  small.
- `[]term.Option` functional-options pattern needs typing. In practice
  only `term.WithKittyKeyboard` is used; a single `enable_kitty: Bool?`
  flag on the `terminal(...)` wrapper would cover the demo cleanly.
- `OnEvent` callback receives `ui.Event` which gets type-asserted to
  `term.EventTitle`, `term.EventNotify`, `term.EventClosed`,
  `term.EventMouseShape`, `term.EventWorkingDirectory`. Either bind
  these as a new Ard type union
  `TerminalEvent = TerminalEventTitle | TerminalEventNotify | …` with
  payload accessors per variant, or expose a flatter
  `(kind: TerminalEventKind, payload: Str)` style dispatch (lossy but
  small surface).
- `widgets/term` is a separate Go sub-package that pulls in its own
  ANSI parser and PTY plumbing (`creack/pty` on Linux/macOS; no
  Windows support). Adds dependency surface and binary size.
- Demo-specific wiring: 8th page slot, hidden from the tab bar but
  reachable via the command palette, plus conditional shortcuts so
  `q` / `n` / `p` don't get swallowed by the shell.

No new layout / styling primitives needed; the only new vaxis APIs
exposed are the terminal widget itself and the per-page-suppression of
shortcuts. Rated **high effort, low API gain** — the rest of the demo
covers the binding surface that exists. Add when there's a concrete
use case beyond demo parity.

---

## Deferred — low demand

### `PollEvent`

```go
func (vx *Vaxis) PollEvent() Event
```

Reads a single event from the channel, blocking until one arrives. The Ard
`read_event()` and `read_key()` methods already block on the same channel.
`PollEvent` would only matter if an application needed to read events from
a goroutine other than the main loop — a pattern that Ard fibers could
enable, but no consumer has asked for yet.

---

## Skipped — cannot express in Ard FFI

### `Options.WithConsole`

```go
type Options struct {
    WithConsole console.Console  // Go interface
}
```

The `console.Console` type is a Go `interface`. Interfaces cannot cross the
Ard FFI boundary. A user who needs this would write their own Go companion.

### `Options.WithTTY`

Passes an arbitrary file path for the TTY device. Expressible in FFI (it is
just a `string`), but not useful on its own without the ability to open and
pass file descriptors, which Ard does not expose.

---

## Skipped — judged unnecessary for now

### `PostEventBlocking`

```go
func (vx *Vaxis) PostEventBlocking(ev Event)
```

Blocks if the event queue is full instead of dropping the event.
**Done** — bound as `raw_post_event_blocking` / `app.post_event_blocking()`.

### `Color.asIndex()` — full vaxis color table

vaxis Go's `asIndex()` uses a 256‑entry lookup with weighted‑distance
matching. Our `index_for_rgb()` uses the standard 6×6×6 cube formula (index
16–231). Both produce perceptually equivalent results for terminal UI work.
The exact table can be copied into the Ard library later with no API
changes.

### Image protocol override

vaxis Go exposes `VAXIS_GRAPHICS` env‑var overrides and protocol constants
(`noGraphics`, `fullBlock`, `halfBlock`, `sixelGraphics`, `kitty`). Our
`new_image()` auto‑selects the best protocol. Manual override would mean an
additional `Options` field or a separate constructor.

### `SyncFunc` event type

vaxis posts `SyncFunc(func())` as an event so the application can call the
function on the main thread. We surface the mechanism through
`app.sync_func(callback)` — the callback is invoked by Go before the call
returns, so the Go event type itself has no Ard representation.

---

## Not applicable

### `Event` (Go empty interface)

```go
type Event interface{}
```

Go's empty interface. In Ard the equivalent is the `Event` type union
(`KeyEvent | ResizeEvent | ...`). The Go empty interface cannot be directly
represented; we encode events as tagged strings across the FFI boundary
instead.

### `capabilities` / `cursorState` / `writer`

Unexported types used internally by the vaxis Go implementation. Not part
of the public API and not needed on the Ard side.
