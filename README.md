# vaxis

Reusable Ard bindings for [Vaxis](https://github.com/rockorager/vaxis).

This package owns both the Ard API surface and the Go FFI companion for the Go target.

## Install

```sh
ard add github.com/akonwi/vaxis-ard@latest
```

Then import it with the package manifest name:

```ard
use vaxis
```

## Basic usage

```ard
use vaxis

fn main() Void {
  let vx = vaxis::new("My TUI").expect("open terminal")
  vaxis::clear(vx)
  vaxis::draw_text(vx, 2, 1, "Hello from Vaxis")
  vaxis::draw_text_style(vx, 2, 3, "Styled", 2, -1, true, false, false, false, false)
  vaxis::render(vx).expect("render")
  vaxis::read_key(vx).expect("read key")
  vaxis::close(vx).expect("close terminal")
}
```

## API surface

### Terminal lifecycle

- `new(title: Str) Vaxis!Str`
- `close(vx: Vaxis) Void!Str`
- `suspend(vx: Vaxis) Void!Str`
- `resume(vx: Vaxis) Void!Str`

### Rendering

- `clear(vx: Vaxis) Void`
- `render(vx: Vaxis) Void!Str`
- `refresh(vx: Vaxis) Void`
- `draw_text(vx: Vaxis, x: Int, y: Int, text: Str) Void`
- `draw_text_style(vx: Vaxis, x: Int, y: Int, text: Str, fg: Int, bg: Int, bold: Bool, dim: Bool, italic: Bool, underline: Bool, reverse: Bool) Void`
- `rendered_width(vx: Vaxis, text: Str) Int`

Colors use `-1` for terminal default, `0..255` for indexed colors, and values above `255` as packed `RRGGBB` RGB integer values.

### Windows

- `root(vx: Vaxis) Window`
- `window(vx: Vaxis, x: Int, y: Int, width: Int, height: Int) Window`
- `subwindow(win: Window, x: Int, y: Int, width: Int, height: Int) Window`
- `window_width(win: Window) Int`
- `window_height(win: Window) Int`
- `window_clear(win: Window) Void`
- `window_draw_text(win: Window, x: Int, y: Int, text: Str) Void`
- `window_draw_text_style(win: Window, x: Int, y: Int, text: Str, fg: Int, bg: Int, bold: Bool, dim: Bool, italic: Bool, underline: Bool, reverse: Bool) Void`
- `window_fill(win: Window, text: Str, fg: Int, bg: Int, bold: Bool, dim: Bool, italic: Bool, underline: Bool, reverse: Bool) Void`
- `window_show_cursor(win: Window, x: Int, y: Int) Void`

### Input and terminal helpers

- `read_key(vx: Vaxis) Str!Str`
- `text_backspace(text: Str) Str`
- `set_title(vx: Vaxis, title: Str) Void`
- `hide_cursor(vx: Vaxis) Void`
- `show_cursor(vx: Vaxis, x: Int, y: Int) Void`
- `bell(vx: Vaxis) Void`
- `notify(vx: Vaxis, title: Str, body: Str) Void`
- `clipboard_push(vx: Vaxis, text: Str) Void`
- `terminal_id(vx: Vaxis) Str`

`read_key` returns normalized app-friendly strings such as `q`, `r`, `up`, `down`, `left`, `right`, `select`, `backspace`, `redraw`, or a printable character.

### Capabilities

- `can_rgb(vx: Vaxis) Bool`
- `can_sixel(vx: Vaxis) Bool`
- `can_kitty_graphics(vx: Vaxis) Bool`
- `can_unicode_core(vx: Vaxis) Bool`
- `can_display_graphics(vx: Vaxis) Bool`

## Examples

Examples live under `examples/`.
