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
  + vaxis/ui bindings              Graphics protocols
  + ui widgets (ui.ard)            vaxis/ui widget FFI (ffi/ui.go)
```

### What belongs in pure Ard
- All type definitions (structs, enums, type unions)
- Widget trait and widget implementations
- Window coordinate math
- Cell buffer manipulation
- Event type definitions and match dispatch
- Application-level helpers and sugar
- vaxis/ui widget constructors and state management

### What belongs in Go FFI
- Terminal open/close (raw mode, alt screen enter/exit)
- Raw stdin reading and ANSI sequence parsing
- Signal handling (SIGWINCH, etc.)
- `select`-based event multiplexing (Ard lacks `select`)
- Final ANSI escape writing to the terminal
- Graphics protocols (sixel, kitty)
- Capability queries (DA1, XTGETTCAP, etc.)
- vaxis/ui widget construction (Go interfaces + closures can't cross FFI directly)

## Files

```
vaxis.ard          # Main library: base types + extern declarations (→ ffi/host.go)
ui.ard             # vaxis/ui bindings: Widget constructors, State<T>, actions, focus
ffi/
  host.go          # Base vaxis Go FFI companion (package ffi)
  ui.go            # vaxis/ui Go wrappers (package ffi)
uitest.ard         # (planned) vaxis/ui/uitest bindings — separate module
examples/
  counter/         # Counter TUI example (imperative, uses vaxis base API)
  tic-tac-toe/     # Tic-tac-toe TUI example (imperative)
  todo/            # Todo list TUI example (declarative, uses vaxis/ui widgets)
```

## Module imports

```ard
use vaxis          # base types: App, Window, Style, Color, Cell, Event, etc.
use vaxis/ui       # ui widgets: text, column, row, stateful, actions, shortcuts, etc.
```

- `vaxis.ard` is the package root module → imported with `use vaxis`
- `ui.ard` is a nested module in the same package → imported with `use vaxis/ui`
- Root module externs use qualified names (`= "vaxis.FnName"`)
- Nested module externs use unqualified names (`= "FnName"`) — they resolve to `package ffi` in the same dependency

## vaxis/ui bindings (complete)

All public vaxis/ui widgets and types are bound. See `ui.ard` and `ffi/ui.go`.

| Category | Widgets / Types |
|---|---|
| Layout | `row`, `column`, `center`, `padding`, `expanded`, `sized_box`, `align`, `constrained_box`, `stack`, `flex`, `positioned` |
| Basic | `text`, `styled_text`, `button`, `divider`, `rich_text`, `span` |
| Input | `text_field`, `text_area`, `checkbox`, `radio`, `segmented_control`, `shortcuts`, `actions`, `action`, `focus`, `focus_scope` |
| Scroll | `scroll_view`, `scrollbar`, `scroll_pane`, `scroll_pane_controller`, `custom_scroll_view`, slivers (`sliver_to_box`, `sliver_pinned_header`, `sliver_fill_remaining`, `sliver_list`) |
| State | `stateful`, `State<T>`, `BuildContext`, `EventContext`, `EventResult` |
| Styling | `decorated_box`, `bordered_box`, `progress_bar` |
| Polish | `dialog`, `modal_barrier`, `list_tile`, `overlay`, `overlay_modal`, `command_palette`, `cmd_item` |
| Selection | `selection_area`, `selection_container` |
| Table | `table`, `table_row`, `table_column_intrinsic`/`_fixed`/`_flex` |
| Cursor | `cursor` (with shape constants) |
| Animation | `animation_controller`, `anim_forward`/`_reset`/`_stop`, `anim_value` |
| Theme | `run_themed` (via `WithBaseColors`) |
| Lifecycle | `run`, `quit` |

### Layout constants

- Flex: `AXIS_HORIZONTAL`/`_VERTICAL`, `MAIN_SIZE_MAX`/`_MIN`, `MAIN_ALIGN_START`/`_END`/`_CENTER`/`_BETWEEN`/`_AROUND`/`_EVENLY`, `CROSS_ALIGN_CENTER`/`_START`/`_END`/`_STRETCH`
- Scroll: `SCROLL_VERTICAL`, `SCROLL_HORIZONTAL`
- Cursor: `CURSOR_DEFAULT`, `CURSOR_BLOCK`/`_BLINKING`, `CURSOR_UNDERLINE`/`_BLINKING`, `CURSOR_BEAM`/`_BLINKING`

### Stateful widget pattern

```ard
fn root_widget() ui::Widget {
  ui::stateful(
    key: "my-key",
    init: fn(ctx: ui::BuildContext) MyState { MyState{...} },
    build: fn(ctx: ui::BuildContext, state: ui::State<MyState>) ui::Widget {
      let model = state.value()
      // ... return widget tree using model ...
    },
  )
}
```

Mutate state from action handlers:
```ard
ui::action("my-intent", fn(ctx: ui::EventContext, intent: Str) ui::EventResult {
  state.set(fn(mut s: MyState) { s.field = new_value })
  ui::EventResult::handled
})
```

## Testing

Python PTY smoke tests exercise built example binaries behind a pseudoterminal.
Each example has a `test_<name>.py` that builds the binary, spawns it with `pty.fork()`,
feeds keystrokes, and asserts on visible screen text.

```bash
python3 examples/todo/test_todo.py    # builds + runs todo smoke test
python3 examples/counter/test_counter.py
python3 examples/tic-tac-toe/test_tic_tac_toe.py
```

Run the relevant smoke test after each round of binding changes to validate
nothing regressed.

## Ard conventions for this project

- Use `ard/result` and `ard/maybe` for error handling and optionality
- Prefer type unions for sum types (e.g. `type Event = KeyEvent | ResizeEvent | ...`)
- Prefer `match` over cascading `if`/`else` for dispatch
- Keep `extern fn` declarations minimal and focused
- Mutating methods use `mut fn` in `impl` blocks
- Public API uses traits for extensibility (e.g. `Widget`, `Drawable`)
- **FFI externs are `private` and prefixed `raw_`** (e.g. `private extern fn raw_open(...)`).
  These are private implementation details. Consumer-facing API wraps them
  in methods on `App`, `Window`, or free functions.
- Examples import the library; they do **not** duplicate extern declarations inline

## Key Ard language notes

- **No `return`**: use `if/else` as final expression or mutable variables for early exit
- **No `select`**: channel multiplexing must happen in Go FFI
- **No `defer`**: cleanup is explicit
- **No bitwise operators**: use `+` for accumulating non-overlapping flags
- **No `::` for static methods**: constructors are free functions, not `impl` methods
- **No struct spread `..self`**: copy every field explicitly
- **No field shorthand**: write `{ grapheme: grapheme }`, not `{ grapheme }`
- **No `let mut`**: mutable bindings are `mut x = ...`, not `let mut x = ...`
- **`mut` params**: `fn foo(mut app: App)` for mutable function parameters
- **Enums don't carry data**: use type unions for discriminated unions
- **Enum match variants must be qualified**: `UnderlineStyle::off`, not bare `off`
- **Type union values are bare structs**: `DefaultColor{}`, not `Color::DefaultColor(...)`
- **Functions must be defined before use** within a file
- **`if/else` as value**: works as final expression but not in `let` bindings
- **No `;` statement separator**: newlines only
- **No `.slice()` on `Str`**: use `for ch in text` for grapheme iteration
- **No generic trait bounds** (yet): unconstrained generics are fine for TUI use cases
- **No pointer types**: use `mut T` for mutable references, `extern type` for opaque FFI handles
- **`list.push()` requires local target**: cannot call `.push()` on a list captured from an enclosing scope (e.g. inside closures). Extract list mutations into free functions that return a new list.
- **`if/else` as value only works as the last expression in a function**: cannot use `let x = if ... { ... } else { ... }`. Use `mut` with separate `if` assignments instead.
- **`Void!Str` returns need `Result::ok(())`**: fallible functions must explicitly wrap success

## Dependencies

- Go: `git.sr.ht/~rockorager/vaxis` v0.16.0
- Ard: `>= 0.19.2`
- Go target requires Go ≥1.25

## Reference

- vaxis Go source: `git.sr.ht/~rockorager/vaxis`
- Ard docs: https://ard.run
- This package on GitHub: `github.com/akonwi/vaxis-ard`
