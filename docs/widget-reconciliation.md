# Widget reconciliation pitfalls

How `vaxis/ui` re-renders a tree when state changes, and one
surprising failure mode that's easy to walk into.

Upstream source references:

- `vaxis/ui/element.go` — element reconciliation (`UpdateChild`,
  `canUpdate`)
- `vaxis/ui/state.go` — `StateBase.MarkNeedsBuild` and the rebuild
  pump
- `vaxis/ui/widget.go` — `canUpdate` (type + key equality)
- `vaxis/ui/app.go` — `pump`, `Paint`, `rootRO`

Repro reference: `examples/reconcile_bug.ard`.

---

## How reconciliation is supposed to work

1. `state.set(...)` mutates the state value and calls
   `MarkNeedsBuild` on the owning stateful element.
2. On the next frame, the rebuild pump processes dirty elements.
3. The stateful's `Rebuild` calls `UpdateChild(oldChild, newWidget)`.
4. `canUpdate(oldWidget, newWidget)` compares **types** via
   `reflect.TypeOf` (and matches widget keys if present):
   - **Same type** → update the existing element in place, then
     `Rebuild` it.
   - **Different type** → unmount the old element, mount a new one,
     then `Rebuild` it.
5. `findRenderObject` walks the new element tree, the runner lays out
   the new root render object, and `Paint` paints it.

So in principle, switching the build output between
`ui::center(ui::text(...))` and `ui::column([...])` should work: the
old `Center` element is unmounted, a new `Column` is mounted, paint
walks the new tree.

---

## The pitfall

In practice, **switching the build output's outer widget type
between frames does not repaint the new tree** (at least in the
shapes we've hit so far).

Concrete shape that breaks:

```ard
build: fn(ctx, state) ui::Widget {
  let model = state.value()
  // body returns ui::center(text) when loading, ui::column(rows) when ready
  ui::shortcuts(
    ui::actions(
      ui::focus_scope(ui::focus(body(model)), trap: false, auto_focus: true),
      [/* … */],
    ),
    [/* … */],
  )
}
```

Where `body` is:

```ard
fn body(model: View) ui::Widget {
  if model.loading {
    ui::center(ui::text("Loading…"))
  } else {
    ui::column(rows_for(model))
  }
}
```

Observed (verified with debug logs at every step):

- The async fetch completes.
- `state.set` runs, the new state has `loading=false` and items in it.
- The build closure fires again with the new state.
- `body` evaluates the `else` branch and returns the `Column` widget.
- `render_list` (or whatever builds the rows) is called.

But the screen keeps showing the `Loading…` text. The new `Column`
tree never appears. Tab-switching away and back has no effect because
the inbox view remounts and starts the loading state over.

## What DOES work

- **Same outer widget type, different content.** Both branches return
  `ui::center(ui::text("…"))`, just with different strings:

  ```ard
  if model.loading {
    ui::center(ui::text("LOADING"))
  } else {
    ui::center(ui::text("LOADED items=" + …))
  }
  ```

  The text updates correctly on rebuild.

- **Same outer widget type, different children list.** Both branches
  return `ui::column([…])`, with different items inside:

  ```ard
  if model.loading {
    ui::column([ui::center(ui::text("Loading…"))])
  } else {
    ui::column(rows_for(model))
  }
  ```

  The column children update correctly on rebuild — this is the
  workaround we ended up using in tinear's inbox view.

- **Unconditional widget tree.** Returning the same expression every
  build (e.g. `ui::column(rows_for(model))`) works — `rows_for` can
  return an empty list initially, and the column is updated as the
  list changes.

---

## Why this is unexpected

Type-changing reconciliation is a **basic** capability of any
retained-mode UI framework — Flutter, React, SwiftUI, and vaxis/ui's
own reconciler all have explicit code paths for it
(`UpdateChild` + `canUpdate=false` → unmount + mount). When the user
view of state is "I rebuild and return whatever widget tree I want",
having that tree silently not paint when its top-level type changes
is genuinely surprising.

## Required preconditions (verified)

We narrowed down the bug shape by running
`examples/reconcile_bug.ard` in two configurations:

1. **Single stateful at the root** — cross-type toggle WORKS. The
   screen correctly flips between `Center(Text)` and `Column([Text])`.
2. **Three nested statefuls** (`outer` wraps `middle` wraps `inner`,
   each with a `Focus(Column([Expanded(…)]))` wrapper around the
   next layer) — cross-type toggle on the innermost stateful FAILS.
   The first frame's tree stays visible.

So the bug requires:

- a dirty stateful element,
- **nested inside one or more ancestor `renderObjectElement`s**
  (e.g. `column`, `row`, `padding`, `decorated_box`, etc.),
- whose build output flips its outermost **render-object** widget
  type.

## Diagnosis

In `vaxis/ui/render.go`, `renderObjectElement.Rebuild` only calls
`syncRenderChildren()` on **itself**, after building its own direct
element children. It does not propagate "my descendant's render
object changed" up the element tree to enclosing
`renderObjectElement`s.

In the nested-statefuls layout, the chain typically looks like:

```
…outerColumnEl       (renderFlex — ancestor RO)
  outerExpandedEl
    middleStatefulEl
      middleFocusEl
        middleColumnEl  (renderFlex — intermediate RO)
          middleExpandedEl
            innerStatefulEl       ← marked dirty by state.set
              innerShortcutsEl
                innerActionsEl
                  innerFocusScopeEl
                    innerFocusEl
                      innerBodyEl ← flips type (Center → Column)
```

When the dispatch fires on `innerStatefulEl` and `innerBodyEl`
switches type:

1. `innerFocus.UpdateChild(oldCenter, newColumnWidget)` resolves
   `canUpdate=false`, unmounts old, mounts new. The new column's
   `Rebuild` syncs its OWN render children (the inner column's rows).
2. **No ancestor is told to re-sync.** `middleColumnEl` and
   `outerColumnEl` (the nearest enclosing `renderObjectElement`s)
   still hold the OLD `renderCenter` reference as their transitive
   render-object child, populated the first time their own
   `syncRenderChildren` ran.
3. At paint time the ancestors walk their stale render-child lists
   and paint the old `renderCenter`. The new `renderFlex` for the
   inner column is orphaned from the render-object tree (mounted as
   an element but not connected upward).

This is consistent with everything else we saw:

- **Same outer type, different content** works because the existing
  render object is mutated in place (no remount), so the ancestor's
  cached pointer still resolves to the correct (updated) RO.
- **Single stateful at the root** works because there's no ancestor
  `renderObjectElement` whose children list can go stale.
- **Nested statefuls** fail because there IS an enclosing
  `renderObjectElement` whose children list is now stale.

## Likely upstream fix

When a `renderObjectElement` re-syncs its children, the new render
objects it adopts (or specifically, the unmount/mount path in
`UpdateChild`) should mark enclosing `renderObjectElement`s dirty for
`syncRenderChildren`, so the ancestors re-discover their transitive
render-object children through `findRenderObject` on the next frame.
A narrower fix would walk back up to the nearest ancestor
`renderObjectElement` immediately on each `UpdateChild` that crosses
a type boundary.

A focused reproducer lives at `examples/reconcile_bug.ard`. Once a
clean upstream issue is filed we'll link it here.

---

## Misdiagnosis trap

This bug presents as "state is updating but the screen never
changes." Because the build closure DOES re-run with fresh state and
the new widget IS constructed, the natural assumption is that
something went wrong on the dispatch/state path — not in painting.

Tinear hit this twice and burned several rounds chasing the wrong
layer both times:

1. The inbox view's initial fetch was moved out of `init` into
   `build` with a comment claiming "cross-module monomorphization
   issues we'd hit if we did it from `init`." That was wrong:
   `init` dispatches were never broken. The real bug was the
   inbox's body returning `ui::center(…)` while loading and
   `ui::column(…)` once loaded — a type swap at the same slot.
   The `init` workaround happened to coincide with another shape
   change that masked the symptom.
2. The issue detail view repeated the pattern (`ui::center` for
   loading, `ui::column` for loaded). Adding debug toasts confirmed
   every step of the dispatch ran. The fix was wrapping the
   conditional in a stable outer widget, not touching the dispatch.

If you find yourself adding probes to a load callback because "state
updates but the screen doesn't," stop and check this first:

- Does the build closure return DIFFERENT outer widget types across
  the state transition?
- Are you nested in at least one ancestor `renderObjectElement`
  (any `column`, `row`, `padding`, `decorated_box`, etc.)?

If yes to both, you're hitting this bug. Don't debug dispatch.

### Sibling bug: stale widget config in `ui::stateful` (FIXED)

The same "state changes but nothing happens" symptom had a totally
different root cause hiding in `uiStatefulState` (`ffi/ui.go`). The
state object cached the widget at `CreateState` time and never
refreshed it. Subsequent rebuilds called `s.widget.Build(...)` —
the ORIGINAL closure, with the ORIGINAL captured locals.

For any parameter the parent passes into `ui::stateful::new(...)`
that the `build:` closure reads (an `active: Bool` flag, an api
key, an `on_open` callback, etc.), the stateful subtree saw the
value captured at first mount and never anything else. Reads from
`ui::stateful`'s own `state` value did update normally because
that goes through `state.value()` / `state.set(...)`, which are
backed by the live `UiStateContext`, not the widget closure.

Fixed by reading `s.StateBase.Widget().(uiStateful[T])` on every
Build instead of using a cached field. If you ever revisit the FFI:
**don't cache widget references on the State** — always read
`StateBase.Widget()` so widget updates propagate.

Surfaced in tinear when only one inner tab view’s focus_scope
responded to keys after a tab change; the `active` flag was frozen
at first mount. If you see a parent-supplied param appearing stale
inside a stateful subtree, suspect this class of bug first.

## Workarounds you can use today

1. **Wrap conditional branches in the same outer widget type.**
   Pick a common outer (often `column`, `decorated_box`, or
   `padding`) and only switch the inner content. This is the
   pattern the demo's `build_page` happens to follow because the
   page is wrapped in `padding(..., all: 1)` at the call site.

2. **Always return the same expression.** Move the conditional
   *inside* the children list:

   ```ard
   ui::column([
     if model.loading {
       loading_widget()
     } else {
       list_widget(model)
     },
   ])
   ```

   The `column` stays the same widget type; its single child
   changes. (Reconciliation also handles this case, but at least it's
   one level deeper than the slot we know breaks.)

3. **Last resort: extract conditional bodies into their own
   stateful widgets with distinct keys** so each branch is its own
   subtree. Less elegant; only reach for it when (1) and (2) aren't
   enough.

---

## Changelog

- 2025-XX: documented the stale-widget-in-`ui::stateful` FFI bug
  (also fixed in this change). Same "state updates but nothing
  happens" symptom, totally different root cause: `uiStatefulState`
  cached the widget at `CreateState` and kept running the original
  build closure. Always read `StateBase.Widget()` instead.


- 2025-XX: added the "Misdiagnosis trap" section after tinear hit
  this a second time on the issue detail view. Recorded that the
  inbox view's `init`-vs-`build` workaround was the wrong fix for
  the same bug, so we don't perpetuate the misdiagnosis.
- 2025-04: initial entry written after reproducing the issue in
  tinear's inbox view. Reproducer landed at
  `examples/reconcile_bug.ard`.
