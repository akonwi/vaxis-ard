# vaxis-ard

Reusable Ard bindings for the [Vaxis](https://github.com/rockorager/vaxis) terminal UI library.

## Goal

The project is pivoting from thin FFI bindings toward a **hybrid port**: the majority of the library lives in idiomatic pure Ard, with a minimal, focused Go FFI companion handling only what Ard cannot do directly. The FFI should be as thin as possible.

## Architecture (target state)

```
Pure Ard (this package)            Go FFI companion (ffi/)
─────────────────────────          ──────────────────────
Types: Style, Color, Cell,         Terminal open/close
  Window, Key, Event               Raw mode + alt screen
Widget trait + implementations     Event multiplexing
Widget library                     (stdin + signals → channel)
Layout helpers                     ANSI escape generation
Application helpers                Capability queries
                                   Graphics protocols
```

### What belongs in pure Ard
- All type definitions (structs, enums, type unions)
- Widget trait and widget implementations
- Window coordinate math
- Cell buffer manipulation
- Event type definitions and match dispatch
- Application-level helpers and sugar

### What belongs in Go FFI
- Terminal open/close (raw mode, alt screen enter/exit)
- Raw stdin reading and ANSI sequence parsing
- Signal handling (SIGWINCH, etc.)
- `select`-based event multiplexing (Ard lacks `select`)
- Final ANSI escape writing to the terminal
- Graphics protocols (sixel, kitty)
- Capability queries (DA1, XTGETTCAP, etc.)

## Files

```
vaxis.ard          # Main library: public types + extern declarations (→ ffi/ host)
ffi/
  host.go          # Go FFI companion (package ffi)
examples/
  counter/         # Counter TUI example
  tic-tac-toe/     # Tic-tac-toe TUI example
  todo/            # Todo list TUI example
```

## Ard conventions for this project

- Use `ard/result` and `ard/maybe` for error handling and optionality
- Prefer type unions for sum types (e.g. `type Event = KeyEvent | ResizeEvent | ...`)
- Prefer `match` over cascading `if`/`else` for dispatch
- Keep `extern fn` declarations minimal and focused
- Mutating methods use `mut fn` in `impl` blocks
- Public API uses traits for extensibility (e.g. `Widget`, `Drawable`)
- **FFI externs are prefixed `_raw_`** (e.g. `_raw_open`, `_raw_render`).
  These are private implementation details. Consumer-facing API wraps them
  in methods on `App`, `Window`, or free functions.

## Key Ard language notes

- **No `select`**: channel multiplexing must happen in Go FFI
- **No `defer`**: cleanup is explicit
- **Enums don't carry data**: use type unions for discriminated unions
- **No generic trait bounds** (yet): unconstrained generics are fine for TUI use cases
- **No pointer types**: use `mut T` for mutable references, `extern type` for opaque FFI handles

## Dependencies

- Go: `git.sr.ht/~rockorager/vaxis` v0.15.0
- Ard: `>= 0.19.2`
- Go target requires Go ≥1.25

## Reference

- vaxis Go source: `git.sr.ht/~rockorager/vaxis`
- Ard docs: https://ard.run
- This package on GitHub: `github.com/akonwi/vaxis-ard`
