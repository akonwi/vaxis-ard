# Binding gaps

vaxis Go API surface not yet exposed in the Ard bindings. Grouped by the
reason they are not implemented.

## Deferred — blocked

### `encode_cells` / `StyledString.encode()`

Serializes `[Cell]` into an ANSI SGR‑encoded string. The implementation is
written and passes as a standalone Ard file, but the Ard compiler panics
(`TypeVar.get` → `$unknown`) when the same code is placed in a library
consumed via `[dependencies]`. The root cause is a compiler bug (the
checker does not recover gracefully from a type‑resolution failure inside a
`match` expression with name‑capture patterns on a type union).

### `ParseStyledString`

Parses an SGR‑annotated string into `[Cell]`. vaxis Go implements this with
its own `ansi` sub‑package. Binding or porting that parser is a substantial
effort. Blocked on having a working `encode_cells` for round‑trip testing,
and on deciding whether to write a pure‑Ard ANSI parser or bind the Go one.

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
