# vaxis-ard examples

Four example TUI programs that exercise the [vaxis-ard](..) bindings.

Each example is a single `.ard` file with its own `fn main()`. They all share
one `ard.toml`, one `go.mod`, and the same FFI companion from the parent
`vaxis-ard` package.

```
counter.ard      – minimal increment/decrement counter
todo.ard         – todo list with inline editing
tic_tac_toe.ard  – tic-tac-toe game
demo.ard         – full vaxis/ui widget showcase (7 pages)
```

## Build

```sh
ard build counter.ard       # → ./counter
ard build todo.ard          # → ./todo
ard build tic_tac_toe.ard   # → ./tic_tac_toe
ard build demo.ard          # → ./demo
```

## Run

```sh
ard run counter.ard
ard run todo.ard
ard run tic_tac_toe.ard
ard run demo.ard
```

## Test

Each example has a Python PTY smoke test that builds the binary, spawns it
under a pseudoterminal, feeds keystrokes, and asserts on visible output:

```sh
python3 test_counter.py
python3 test_todo.py
python3 test_tic_tac_toe.py
python3 test_demo.py
```

## Controls

### counter
- `up` / `right` / `k` / `l` / `+`: increment
- `down` / `left` / `j` / `h` / `-`: decrement
- `r`: reset
- `q` or `Ctrl+C`: quit

### todo
- `up` / `down` or `k` / `j`: move selection
- `enter`: toggle selected item
- `e`: edit selected item
- `a`: add a new empty todo and start editing it
- `d`: delete selected item
- while editing: type to insert, `backspace` to delete, `enter` to save, `q` to cancel
- `r`: reset sample todos
- `q` or `Ctrl+C`: quit

### tic-tac-toe
- arrows or `h` / `j` / `k` / `l`: move selection
- `1`–`9`: jump to a square
- `enter` / `space`: play selected square
- `r`: restart
- `q` or `Ctrl+C`: quit

### demo
- `n` / `p`: next / previous page
- `Tab`: move focus
- `Alt+K`: command palette
- `Alt+P`: profile overlay
- `q`: quit
