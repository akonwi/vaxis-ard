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
vaxis.ard          # Base library: types + extern declarations (→ ffi/host.go)
ui.ard             # vaxis/ui bindings: Widget constructors, State<T>, actions, focus
ffi/
  host.go          # Base vaxis Go FFI companion (package ffi)
  ui.go            # vaxis/ui Go wrappers (package ffi)
uitest.ard         # (planned) vaxis/ui/uitest bindings — separate module
docs/              # In-depth notes on non-obvious runtime behaviour
  README.md        #   Index
  events-and-focus.md  # Event dispatch, Shortcuts/Actions, focus pitfalls
examples/
  counter.ard      # Counter TUI (imperative, uses vaxis base API)
  tic_tac_toe.ard  # Tic-tac-toe TUI (imperative)
  todo.ard         # Todo list TUI (declarative, uses vaxis/ui widgets)
  demo.ard         # Full vaxis/ui widget showcase
  test_harness.py  # Shared PTY test infrastructure
  test_counter.py  # Counter smoke test
  test_tic_tac_toe.py
  test_todo.py
  test_demo.py
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
| Cursor | `cursor` (with `CursorShape` enum) |
| Animation | `animation_controller`, `anim_forward`/`_reset`/`_stop`, `anim_value` |
| Theme | `theme_default`, `must_have_theme`, `provider_theme`, `run(root, theme_set?)` |
| Lifecycle | `run`, `quit` |

### Layout enums (typed)

- Flex: `Axis::horizontal`/`vertical`, `MainSize::max`/`min`, `MainAlign::start`/`end`/`center`/`between`/`around`/`evenly`, `CrossAlign::center`/`start`/`end`/`stretch`
- Scroll: `ScrollAxis::vertical`/`horizontal`
- Cursor: `CursorShape::default`/`block_blinking`/`block`/`underline_blinking`/`underline`/`beam_blinking`/`beam`

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
Shared infrastructure lives in `examples/test_harness.py`.

```bash
python3 examples/test_counter.py    # builds + runs counter smoke test
python3 examples/test_tic_tac_toe.py
python3 examples/test_todo.py
python3 examples/test_demo.py
```

Run the relevant smoke test after each round of binding changes to validate
nothing regressed. The Screen emulator in the harness handles sequential
(imperative) rendering fully; widget-based (vaxis/ui) apps are smoke-tested
for startup, navigation, and clean exit since the emulator doesn't track
vaxis/ui's diff-based cursor positioning.

## API design principles

These guide every new binding and refactor. When a change conflicts with one,
flag it explicitly in the PR description.

### 1. One configurable function, not many single-purpose variants

Do **not** ship `row_end`, `row_center`, `column_stretch`, `flex_stretch`,
etc. Instead ship one `row` / `column` / `flex` that takes optional knobs:

```ard
fn row(
  children: [Widget],
  main_align: MainAlign?,
  cross_align: CrossAlign?,
  main_size: MainSize?,
) Widget
```

Callers omit trailing nullable args they don't care about. This keeps the
public surface small and predictable as more knobs get exposed.

### 2. Typed Ard enums over raw `Int` constants for option values

Do **not** expose `let MAIN_ALIGN_CENTER = 2` style constants in the public
API. Define an Ard enum and convert to int at the FFI boundary:

```ard
enum MainAlign { start, end, center, between, around, evenly }

impl MainAlign {
  fn to_int() Int {
    match self {
      MainAlign::start  => 0,
      MainAlign::end    => 1,
      MainAlign::center => 2,
      // ...
    }
  }
}
```

The vaxis numeric encoding stays an FFI implementation detail. Callers get
compile-time safety and can't pass a meaningless integer.

### 3. Nullable params + `.or(default)` for ergonomic defaults

For any binding with optional configuration, prefer:

```ard
fn rich_text(spans: [TextSpan], soft_wrap: Bool?) Widget {
  _rich_text(spans, soft_wrap.or(false))
}
```

Over pattern A (`match` with `_ => {}`) or pattern B (`if is_some { … }`
followed by manual assignment). `.or(default)` is the idiomatic unwrap
when all you want is a fallback.

Trailing nullable args can be omitted at the call site; non-trailing
omission requires labelled args.

### 4. Public API in Ard-native types; conversion at the FFI boundary

The externs (private, underscore-prefixed) take whatever shape vaxis
needs — ints for enums, opaque handles for Go structs, etc. The public
wrapper does the conversion:

```ard
extern fn _flex(axis: Int, main_size: Int, ..., children: [Widget]) Widget = "UiFlex"

fn flex(axis: Axis, children: [Widget], ...) Widget {
  _flex(axis.to_int(), ..., children)
}
```

The FFI shape is never the user-facing shape.

### 5. Match vaxis defaults at the boundary unless there's a clear TUI reason not to

When filling defaults inside a wrapper, prefer the upstream vaxis zero
value. Deviations are allowed but must be documented in a code comment
right above the default. Example:

```ard
// column defaults to CrossAlign::stretch (TUI columns almost always
// want children stretched to full width).
fn column(...) Widget {
  _flex(..., cross_align.or(CrossAlign::stretch).to_int(), ...)
}
```

### 6. Drop legacy variants when refactoring; this library is not yet published

No deprecation periods, no aliasing the old names. Delete and migrate
call sites in the same change. Examples in `examples/` are part of the
refactor.

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

## Ard language resource: the `ard-expert` sub-agent

When you need authoritative answers about Ard syntax, semantics, stdlib
(`ard/maybe`, `ard/async`, `ard/duration`, …), or idiomatic patterns,
delegate to the **`ard-expert`** sub-agent. It is docs-first for surface
language / type-system / stdlib questions, source-first for compiler
and runtime internals, and cites the relevant page or file in every
reply.

Use it before:
- writing a new extern signature that crosses the FFI boundary,
- introducing a stdlib type you haven't used here before
  (`Fiber`, `Maybe`, `Result`, …),
- adopting a syntax you're unsure compiles (nullable args, enum
  variants, generics, mutable captures, etc.),
- proposing a non-trivial refactor of the public API.

Also useful for **code review** on an Ard change — ask it to sanity-
check a snippet against current language rules.

The "Key Ard language notes" below are a fast cheat-sheet, not a
spec. Anything subtle should be confirmed with `ard-expert`.

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
- **Nullable callback gotcha — `fn(X) Void?` is NOT a nullable function**: the `?` binds to the return type, yielding `fn(X) Maybe[Void]` (a non-null function returning nullable Void). For a *nullable* callback, **omit the return type entirely**: write `on_pressed: fn(EventContext)?`. The parser only treats `?` as nullable on the function type when there's no return type to attach to (`fn(...)?` works; `(fn(...) T)?` grouping is not supported).
- **No `!=` operator**: the lexer recognises `!=` but the parser doesn't — it silently fails with a generic parse error far from the bad token. Use `not (x == y)`. Parentheses required; `not` is the unary-precedence negation keyword (there is no prefix `!`; that means `Result` in *type* position).
- **List indexing is `list.at(i)`, not `list[i]` or `list.get(i)`**: returns the inner `T` (non-nullable) and *panics* on out-of-bounds. There is no subscript syntax in expression position (`[...]` is the list-literal constructor only). Bounds-check with `list.size()` first if `i` isn't statically known to be valid. Related list methods: `.size()`, `.push(v)`, `.prepend(v)`, `.set(i, v)`, `.sort(cmp)`.
- **Closures capture `mut` locals by reference**: a closure created inside `while mut i; … i = i + 1` shares the *live* `i` with every other closure and with the loop. By the time you invoke any of them, `i` is at its post-loop value. Freeze with an immutable `let` inside the loop body, then capture that: `let frozen = i; on_pressed: fn(ctx) { state.set(… s.page = frozen …) }`. `let` bindings are captured by value. Symptom in the demo: all tabs were silently setting `page = 6` and routing to Home via the catch-all, so clicks looked broken.
- **Prefer `for i in range { … }` over `while mut i; … i = i + 1`**: `for` introduces an immutable per-iteration `i` (captured by value), eliminating the loop-closure trap above. Use the `while`-counter form only when you genuinely need a mutable cursor that survives across iterations.
- **Range syntax `a..b` is INCLUSIVE on both ends** (no `..=`, no exclusive variant). `for i in 0..n` iterates `0,1,…,n` — that's `n+1` iterations. For exactly `n` iterations use `for i in 0..(n - 1)`. The bare-int sugar `for i in n` desugars to `0..n` and has the same off-by-one footgun.
- **Cross-file references within a package require a stem import**: a nested module (e.g. `ui.ard` inside the `vaxis` package) cannot `use vaxis` (resolver rejects same-package import via the package name alone) and cannot reference siblings without `use`. To pull a type from `vaxis.ard` into `ui.ard`, write `use vaxis/vaxis` (the file stem) and reference it as `vaxis::Type::variant`. Cleaner long-term: move shared types into a third file (e.g. `style.ard`) that both root and nested modules import.

## Dependencies

- Go: `go.rockorager.dev/vaxis` (post-v0.16.0 tip; pinned via go.mod
  `replace` to a local checkout when consuming unreleased features)
- Ard: `>= 0.19.2`
- Go target requires Go ≥1.26

## In-depth docs

The `docs/` directory captures behaviour and pitfalls that aren't
obvious from the bindings or the upstream surface. Consult it (and add
to it) whenever you debug something subtle that future-you would have
appreciated being told up front.

- [docs/events-and-focus.md](./docs/events-and-focus.md) — event
  dispatch model (capture/target/bubble), how `Shortcuts` and
  `Actions` cooperate via `ctx.Invoke`, the focus registry and
  `focus_scope` semantics, and the most common gotcha: **why inner
  `Shortcuts` cannot rebind `Tab` / `Shift+Tab` / `Escape`** (and the
  intent-hijack workaround).

When you discover another non-obvious interaction, add a focused doc
in `docs/` and link it from `docs/README.md` and from this section.

## Reference

- vaxis Go source: `go.rockorager.dev/vaxis`
  (formerly `git.sr.ht/~rockorager/vaxis` — module path was renamed)
- Ard docs: https://ard.run
- This package on GitHub: `github.com/akonwi/vaxis-ard`
