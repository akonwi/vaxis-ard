# vaxis

Ard bindings for the [Vaxis](https://github.com/rockorager/vaxis) terminal UI library.

Two complementary APIs:
- **`use vaxis`** — base terminal layer: `App`, `Window`, `Style`, `Color`, `Cell`, `Event`
- **`use vaxis/ui`** — Flutter-style widget framework: `stateful()`, `text()`, `row()`, `column()`, `button()`, and ~40 more

## Install

```sh
ard add github.com/akonwi/vaxis-ard@latest
```

## Quick start — vaxis/ui (recommended)

```ard
use vaxis
use vaxis/ui

fn main() Void {
  ui::run(ui::text("Hello from vaxis/ui"))
}
```

Stateful counter:

```ard
use vaxis
use vaxis/ui

struct Counter { count: Int }

fn main() Void {
  ui::run(
    ui::stateful(
      key: "counter",
      init: fn(ctx: ui::BuildContext) Counter { Counter{ count: 0 } },
      build: fn(ctx: ui::BuildContext, state: ui::State<Counter>) ui::Widget {
        let model = state.value()
        ui::shortcuts(
          ui::actions(
            ui::center(ui::text(model.count.to_str())),
            [
              ui::action("inc", fn(ctx: ui::EventContext, intent: Str) ui::EventResult {
                state.set(fn(mut s: Counter) { s.count = s.count + 1 })
                ui::EventResult::handled
              }),
              ui::action("quit", fn(ctx: ui::EventContext, intent: Str) ui::EventResult {
                ui::quit(ctx)
                ui::EventResult::handled
              }),
            ],
          ),
          ["+": "inc", "q": "quit"],
        )
      },
    )
  )
}
```

## Quick start — base vaxis (imperative)

```ard
use vaxis

fn main() Void {
  mut app = vaxis::app_open("My TUI").expect("open terminal")
  let mut win = app.root_window()
  win.clear()
  win.write(2, 1, "Hello from Vaxis", vaxis::default_style())
  try app.render()
  let key = try app.read_key()
  app.close().expect("close terminal")
}
```

## Base vaxis API

### App lifecycle

```ard
mut app = vaxis::app_open(title: Str, opts: Options?).expect("open terminal")
app.close().expect("close terminal")
app.suspend().expect("suspend")
app.resume().expect("resume")
```

### Rendering

```ard
app.render().expect("render")     // flush cell buffer to terminal
app.refresh()                      // force full redraw
let mut win = app.root_window()   // full-screen window
let sub = app.window(col, row, w, h)  // sub-window
```

### Window drawing

```ard
win.clear()                        // clear window region
win.write(col, row, text, style)   // write styled text at position
win.set_cell(col, row, cell)       // place a single Cell
win.fill(cell)                     // fill entire window with a Cell
win.print(col, row, [seg(...)])    // print Segments with wrapping
win.show_cursor(col, row, style?)  // position hardware cursor
```

### Colors and styles

```ard
let red = vaxis::Color::IndexedColor(IndexedColor{ value: 1 })
let blue = vaxis::rgb(0, 0, 255)
let style = vaxis::default_style().with_fg(red).with_bold(true)
```

### Events

```ard
let event = try app.read_event()
match event {
  vaxis::KeyEvent(ev) => { /* ev.key is "q", "up", "a", etc. */ },
  vaxis::ResizeEvent(ev) => { /* ev.cols × ev.rows */ },
  vaxis::MouseEvent(ev) => { /* ev.button, ev.col, ev.row */ },
  vaxis::QuitEvent => { /* quit requested */ },
  _ => {},
}
```

### Input helpers

```ard
let key = try app.read_key()       // blocking, returns "q", "up", etc.
app.set_mouse_shape(vaxis::MouseShape::pointer)
app.post_event("custom-event")
```

## vaxis/ui API

See `ui.ard` for the full binding surface. Key entry points:

| Category | Functions |
|---|---|
| **Lifecycle** | `run(root, theme_set?)`, `quit(ctx)` |
| **Layout** | `row(children, ...)`, `column(children, ...)`, `flex(axis, children, ...)`, `center(child)`, `padding(all, child)`, `expanded(child)`, `sized_box(w, h, child)`, `constrained_box(child, ...)`, `stack(children, ...)`, `positioned(left, top, child)` |
| **Text** | `text(value, ...)`, `styled_text(value, fg?, bg?, attrs?)`, `rich_text(spans, ...)`, `span(text, ...)` |
| **Controls** | `button(label, on_pressed)`, `text_field(value, on_changed, ...)`, `text_area(value, on_changed, ...)`, `checkbox(checked, label, on_changed, ...)`, `radio(value, group, label, on_changed, ...)` |
| **Scroll** | `scroll_view(child, axis?)`, `scrollbar(child, axis?)`, `scroll_pane(child, controller?)`, `custom_scroll_view(controller, slivers, ...)`, `sliver_list_builder(count, builder, ...)` |
| **State** | `stateful(key, init, build)`, `State<T>`, `BuildContext`, `Runtime<T>` |
| **Input** | `shortcuts(child, bindings)`, `actions(child, bindings)`, `action(name, handler)`, `focus(child)`, `focus_scope(child, trap, auto_focus)` |
| **Overlays** | `dialog(title, child, actions, width, on_dismiss)`, `overlay(child, entries)`, `overlay_modal(child, modal_child)`, `command_palette(items, ...)` |
| **Tables** | `table(columns, column_gap, row_gap, rows)`, `table_column_intrinsic()`, `table_column_fixed(w)`, `table_column_flex(f)` |
| **Theme** | `theme_default()`, `must_have_theme(ctx)`, `provider_theme(theme, child)`, `default_theme_set()` |
| **Animation** | `animation_controller(ctx, duration_ms, ...)`, `anim_forward(ctrl)`, `anim_reset(ctrl)`, `anim_value(ctrl)` |

## Examples

```bash
cd examples/

# Imperative (base vaxis)
ard build --out counter counter.ard && ./counter
ard build --out tic_tac_toe tic_tac_toe.ard && ./tic_tac_toe

# Declarative (vaxis/ui widgets)
ard build --out todo todo.ard && ./todo
ard build --out demo demo.ard && ./demo
```

## Testing

Python PTY smoke tests exercise built example binaries behind a pseudoterminal.

```bash
cd examples/
python3 test_counter.py
python3 test_tic_tac_toe.py
python3 test_todo.py
python3 test_demo.py
```

Infrastructure is shared in `test_harness.py`. Run tests after binding changes to validate nothing regressed.

## Dependencies

- Go: `go.rockorager.dev/vaxis` (post-v0.16.0 tip)
- Ard: `>= 0.19.2`
- Go target requires Go ≥1.26
