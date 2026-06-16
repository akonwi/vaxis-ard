# vaxis-ard docs

In-depth notes on how the vaxis-ard bindings (and the upstream `vaxis/ui`
runtime) actually behave. The cheat-sheet in `AGENTS.md` covers basic
patterns; this directory is for the subtleties that you only find when
something doesn't work.

## Index

- [events-and-focus.md](./events-and-focus.md) — event dispatch (capture
  / target / bubble), `Shortcuts`, `Actions`, intent dispatch via
  `ctx.Invoke`, focus widgets, app-level default shortcuts, and the
  common pitfalls (e.g. why you can't rebind `Tab` from an inner
  `Shortcuts`).

## Conventions

- Each doc states what it covers and what upstream vaxis source files
  it depends on.
- When upstream behaviour changes, update the doc *and* link the
  upstream commit.
- Examples reference real binding code (`ui.ard`, `ffi/ui.go`) and real
  apps (the `examples/` directory, or downstream tinear) so the docs
  don't drift from runnable code.
