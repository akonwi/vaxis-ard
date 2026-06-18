# Events, shortcuts, actions, and focus

How key events flow through a `vaxis/ui` app, how `Shortcuts` and
`Actions` cooperate via intents, where focus lives, and the
non-obvious interactions between them. If a keyboard binding or focus
behaviour isn't working the way you expect, the answer is almost
always in this file.

Upstream source references:
- `vaxis/ui/app.go` — root widget construction, event dispatch,
  focus registry
- `vaxis/ui/shortcuts.go` — `Shortcuts` widget + `DefaultShortcuts()`
- `vaxis/ui/actions.go` — `Actions`, `DefaultActions`
- `vaxis/ui/intents.go` — `IntentType`, `NextFocusIntent`,
  `PreviousFocusIntent`, `DismissIntent`, `ActivateIntent`
- `vaxis/ui/focus.go` — `Focus`, `FocusScope`, `FocusOptions`
- `vaxis/ui/event.go` — `EventContext`, `Invoke`

---

## 1. The app root is not your root

When you call `ui::run(root, theme_set)`, the runtime wraps your tree:

```
Actions {
  Bindings: {
    "vaxis.next-focus":     ctx.FocusNext(),
    "vaxis.previous-focus": ctx.FocusPrevious(),
  },
  Child: Shortcuts {
    Bindings: DefaultShortcuts(),   // see below
    Child:    Provider[Theme]{
      Child: <your root>,
    },
  },
}
```

`DefaultShortcuts()`:

| Key         | Intent                     |
|-------------|----------------------------|
| `Tab`       | `NextFocusIntent` (`"vaxis.next-focus"`)         |
| `Shift+Tab` | `PreviousFocusIntent` (`"vaxis.previous-focus"`) |
| `Escape`    | `DismissIntent` (`"vaxis.dismiss"`)              |

These wrappers exist *above* your root. They get to process events
before any widget you write.

Implication: **any key in `DefaultShortcuts()` cannot be rebound by a
`Shortcuts` you nest inside your root.** See §5 for the workaround.

To replace the app-level shortcut map entirely, upstream supports
`ui.WithShortcuts(map)` as a `ui.NewApp` option. The vaxis-ard binding
doesn't currently surface that — open an issue if you need it.

---

## 2. Event dispatch path

For each non-mouse event the runtime:

1. Picks the **target** = the currently focused element
   (`a.focused.element`).
2. Builds a `path` from root → target (`a.pathTo(target)`).
3. Calls `dispatchPath(path, ev)`, which runs the event in three
   phases:

```
capture phase:  path[0]  → path[len-2]    (root → focused.parent)
target phase:   path[len-1]                (focused itself)
bubble phase:   path[len-2] → path[0]      (focused.parent → root)
```

Each step calls `element.HandleEvent(ctx, ev)`. The **first widget to
return `EventHandled` stops the dispatch.**

Mouse events use a hit-test path instead, but the phase model is the
same.

Important: **capture descends from the root**, so anything in your
tree that sits *above* a widget is checked *before* that widget.

---

## 3. `Shortcuts`: keys → intents

`Shortcuts.HandleEvent` is phase-insensitive. It runs in every phase,
including capture. It looks like:

```go
for binding, intent := range w.Bindings {
    if key.MatchString(binding) && ctx.Invoke(intent) == EventHandled {
        return EventHandled
    }
}
return EventIgnored
```

Two things to internalise:

- A `Shortcuts` only *emits* the intent. It does not *handle* the key.
  The intent has to be picked up by an `Actions` somewhere, or the
  `Shortcuts` returns `Ignored` and the event keeps flowing.
- Because `Shortcuts` doesn't filter by phase and capture is first,
  the **outermost matching `Shortcuts` wins** for any given key. An
  inner `Shortcuts` further down the tree may never see the event.

---

## 4. `Actions`: intents → handlers (walks UP from target)

`ctx.Invoke(intent)` from `vaxis/ui/event.go`:

```go
target := c.target          // the focused element
intentType := intent.IntentType()
for e := target; e != nil; e = e.Base().parent {
    if actions, ok := e.(actionProvider); ok {
        action, ok := actions.action(intentType)
        if ok {
            return runAction(action, c, intent)
        }
    }
    // ... DefaultActions are remembered as a fallback ...
}
```

Key points:

- The walk starts at the **focused element** (the event target), not
  at the `Shortcuts` widget that called `Invoke`.
- The **first** matching `Actions` wins — closer to the focused widget
  beats farther away.
- A handler that returns `EventIgnored` is still considered "found"
  and short-circuits the walk. Upstream `Actions` will not be tried
  after a closer handler returns ignored.
- `DefaultActions` are remembered as a fallback only if no regular
  `Actions` handler is found at all.

This is why inner `Actions` can "intercept" an upstream intent
(see §5). It's also why placement of `Actions` matters: it must be on
the ancestor chain of whatever widget actually holds focus.

---

## 5. Pitfall: inner `Shortcuts` cannot rebind `Tab` / `Shift+Tab` / `Escape`

This is the single most surprising interaction in the runtime.

### What you tried

```ard
ui::shortcuts(
  ui::actions(
    body,
    [
      ui::action("my.next-tab", fn(_ctx, _) { … }),
    ],
  ),
  ["Tab": "my.next-tab"],
)
```

### Why it doesn't fire

1. App-level `Shortcuts` (above your root) is checked **first in
   capture phase** because capture descends from root.
2. It matches `Tab` → emits `NextFocusIntent`.
3. `Invoke` walks UP from the focused widget, finds the app-level
   `Actions` (since nothing closer handles `vaxis.next-focus`), and
   calls `FocusNext()`.
4. App-level `Shortcuts` returns `EventHandled`. Dispatch ends. Your
   inner `Shortcuts` is never reached.

### The fix: hijack the intent, not the key

Instead of trying to remap `Tab`, install a handler for the upstream
intent in your own `Actions`. Because `Invoke` walks UP from the
focused widget, your closer `Actions` wins over the app-level one:

```ard
ui::actions(
  ui::focus_scope(
    ui::focus(body),
    trap: false,
    auto_focus: true,
  ),
  [
    ui::action("vaxis.next-focus", fn(_ctx: ui::EventContext, _intent: Str) ui::EventResult {
      // your "next tab" behaviour
      ui::EventResult::handled
    }),
    ui::action("vaxis.previous-focus", fn(_ctx: ui::EventContext, _intent: Str) ui::EventResult {
      // your "previous tab" behaviour
      ui::EventResult::handled
    }),
    ui::action("vaxis.dismiss", fn(_ctx: ui::EventContext, _intent: Str) ui::EventResult {
      // your "Esc" behaviour; return Ignored to let it fall through
      ui::EventResult::handled
    }),
  ],
)
```

For non-default keys (e.g. `1`, `2`, `?`, `r`) you still want a
`Shortcuts` of your own with your custom intent strings.

### Caveat

Hijacking `vaxis.next-focus` overrides focus traversal for your whole
subtree. That's fine when there's only one focusable inside (e.g. a
tab-bar shell). When your subtree contains multiple focusables that
should still be Tab-traversable, restrict the hijack to a region (use
`Actions` lower in the tree, or use `focus_scope(trap: true)` to keep
upstream `FocusNext` cycling within a sub-region).

---

## 6. Focus widgets

Two distinct widgets, often confused.

### `ui::focus(child)`

Wraps `child` as a **focus target**. The element registers itself with
the app's focus registry on mount.

```ard
ui::focus(my_widget)
```

The app auto-focuses the *first* registered focusable when nothing is
currently focused. So a single bare `ui::focus(...)` will often "just
work" on initial mount — but its position in registration order
matters, and stale focus state from an earlier subtree can defeat it.

`ui::focus(...)` does **not** scope or trap focus. Tab traversal walks
through every focus target in the app, in registration order.

### `ui::focus_scope(child, trap, auto_focus, reclaim_focus?)`

A *scope* around a region of the tree. Not itself a focus target;
needs `ui::focus(...)` widgets inside it.

```ard
ui::focus_scope(
  ui::focus(inner),
  trap: true,           // keep Tab traversal inside this subtree
  auto_focus: true,     // pull focus into the scope when it mounts
  reclaim_focus: true,  // re-pull on every rebuild while focus is outside
)
```

- `auto_focus: true` calls `focusFirstWithin(scope)` on mount,
  guaranteeing focus lands somewhere inside the scope even if focus
  state was weird coming in (e.g. previous screen had focus on a
  field that just unmounted).
- `trap: true` keeps `Tab` / `Shift+Tab` cycling inside the scope.
  Useful for modal dialogs and overlay surfaces.
- `reclaim_focus: true` re-runs the auto_focus pass on every rebuild
  whenever focus is currently outside the scope. Without it,
  `auto_focus` only ever fires once. The flag matters specifically
  when the focused element may be **disposed mid-life** — see §10.
  Defaults to `false` (omit the argument). Mirrors
  `FocusScope.ReclaimFocus` upstream.

### Pitfall: an outer `ui::focus` steals focus from an inner subtree

```ard
ui::focus(
  ui::stateful(
    …,
    build: fn(…) { build_one_of_several_screens(…) },
  ),
)
```

The outer `ui::focus(...)` registers a focus target *above* every
screen. When the app starts, the outer focus is the first registered
focusable and grabs initial focus. Key events bubble from that outer
focus upward — they **never pass through any `Shortcuts` / `Actions`
that live deeper inside the stateful subtree.** So screen-level
shortcuts silently don't fire.

Fix: drop the outer `ui::focus(...)`. Let each screen install its own
focus target(s). Wrap screens that need a guaranteed focus landing in
`ui::focus_scope(..., auto_focus: true)`.

---

## 7. Default intents reference

From `vaxis/ui/intents.go`:

| `IntentType` (string)        | Type                   | Default app-level handler |
|------------------------------|------------------------|---------------------------|
| `"vaxis.activate"`           | `ActivateIntent`       | n/a (Button etc. register `DefaultActions`) |
| `"vaxis.dismiss"`            | `DismissIntent`        | n/a (Dialog etc. register `DefaultActions`) |
| `"vaxis.next-focus"`         | `NextFocusIntent`      | `ctx.FocusNext()` |
| `"vaxis.previous-focus"`     | `PreviousFocusIntent`  | `ctx.FocusPrevious()` |
| `"vaxis.toggle-profile-overlay"` | `ToggleProfileOverlayIntent` | toggles the profile overlay |

You can install handlers for any of these strings in your own
`Actions`. The earliest match on the focused-to-root walk wins.

---

## 8. Mental model checklist

When a key isn't doing what you expect, work the list:

1. **Is the right widget focused?** Add a temporary log in your
   `Actions` handler (intent string is the second arg) or check that
   `ui::focus(...)` / `ui::focus_scope(..., auto_focus: true)` is
   actually mounted where you think it is.
2. **Is there an outer `ui::focus(...)` swallowing initial focus?**
   See §6. Remove it.
3. **Is the key in `DefaultShortcuts()`?** (`Tab` / `Shift+Tab` /
   `Esc`.) If yes, see §5 — bind the upstream intent in your
   `Actions`, not the key in a nested `Shortcuts`.
4. **Is your `Actions` on the ancestor chain of the focused
   element?** If not, `ctx.Invoke` will never visit it.
5. **Did a closer `Actions` return `Ignored`?** Closer wins even when
   it ignores — your further-out handler will never be tried.
6. **Phase confusion**: there is none in `Shortcuts` / `Actions`.
   They run in every phase. The "outer wins" effect comes purely from
   capture descending from the root.
7. **Did a focused widget just unmount?** (modal dismissed, dynamic
   tab closed, list item removed.) Symptom: "first keypress after
   dismiss does nothing" or "nothing responds." See §9.

---

## 9. Focus restoration after a focused widget unmounts

When the currently-focused element is removed from the tree (modal
closed, dynamic tab dismissed, list pruned, screen swapped), focus
has to go *somewhere*. Two mechanisms handle this. Understanding both
is the difference between "first keypress is lost" / "nothing
responds" and "it just works."

### Mechanism A: vaxis's built-in `pendingFocusFallback`

In `vaxis/ui/app.go`, when `unregisterFocusTarget` runs *during a
build* (which is when modal closes happen — a state change triggers
the rebuild that unmounts the modal subtree), it saves the disposed
element's index, debug ID, and label, then sets
`pendingFocusFallback = true`:

```go
if a.build.building {
    a.pendingFocusFallback = true
    a.pendingFocusFallbackIndex = removed
    a.pendingFocusFallbackID    = id
    a.pendingFocusFallbackLabel = label
    return
}
```

At the end of `BuildScope()`, `resolvePendingFocusFallback()` tries
(in order) to focus:

1. a remaining focusable with the same debug **ID**,
2. a remaining focusable with the same debug **label**,
3. the focusable at the disposed element's **index** (clamped to the
   valid range).

For upstream `Dialog` / `CommandPalette`, step 3 typically lands on a
sensible neighbour — frequently the focusable that opened the modal
(e.g. the button that was right before the dialog in registration
order). That neighbour's `Actions` / `Shortcuts` ancestor chain is the
same one that was handling keys before the modal opened, so
navigation "just works" after dismiss.

The demo (`examples/demo.ard`) relies entirely on this mechanism for
dialog dismissal. It never adds `reclaim_focus` to anything in its
own subtree.

### Mechanism B: `ReclaimFocus` on a `FocusScope` you own

If the index-based fallback is unreliable for your layout (see
pitfalls below), you can put a `focus_scope(..., reclaim_focus: true)`
around a *specific* subtree and it will explicitly pull focus back
into that subtree on the same build cycle. It beats
`resolvePendingFocusFallback` to the punch — `focusFirstWithin` sets
the focused element, so the late fallback sees
`focused.element != nil` and no-ops.

Upstream `Dialog` already uses this internally:

```go
// vaxis/ui/dialog.go
Child: FocusScope{
    Trap:         true,
    AutoFocus:    true,
    ReclaimFocus: !w.DisableFocusReclaim,   // true by default
    Child:        ...
},
```

That's why focus moves *into* a dialog on open and stays trapped
there even across rebuilds — you don't have to do anything.

### When the index fallback isn't enough

The fallback picks *some* remaining focusable. For it to be useful,
events from that focusable must bubble through the **right**
`Actions` / `Shortcuts` chain (the one that handles the keys you
care about). Two common patterns break this:

1. **Nested `Shortcuts` / `Actions` layers.** When different
   subtrees install handlers for different intents (e.g. a board
   view binds `j`/`k`/`s`/`y` only when focus is inside the board
   subtree), the fallback landing on a sibling focusable (a tab
   chip, a focusable in an inactive tab subtree, etc.) means events
   bubble through *its* ancestors and never reach the board's
   handlers.

2. **Outer `ui::focus(...)` wrappers sitting above the handler
   layers** (e.g. wrapping a top-level screen in `ui::focus(...)` so
   `focus_scope`'s `auto_focus` always has a target). These are
   tree-order-early focusables; the index fallback can land on them,
   and events from them bubble up *past* the inner shortcuts you
   meant to fire.

In both cases the symptom looks like "first nav press after dismiss
does nothing" (the fallback landed somewhere your handlers don't see)
or "nothing responds at all" (a higher reclaim_focus pinned focus on
an even less useful target). The fix is mechanism B: put
`reclaim_focus: true` on the **innermost scope whose handlers should
fire** after the dismiss. That scope's rebuild explicitly snaps focus
back into its own subtree, so events bubble through the right chain.

### Don't add `reclaim_focus` to every scope

It's tempting to slap `reclaim_focus: true` on every nested
`focus_scope` just in case. Don't — it has a real failure mode.

`reclaim_focus` fires on every rebuild whenever `focusedWithin` is
false. If you have two sibling scopes (e.g. two tabs kept mounted by
`indexed_stack`), both with `reclaim_focus: true`, and focus is
currently in scope A:

- Scope B's rebuild: `focusedWithin(B) = false` → runs
  `focusFirstWithin(B)` → focus moves to B.
- Scope A's rebuild: `focusedWithin(A) = false` (it's now in B) →
  runs `focusFirstWithin(A)` → focus moves back to A.

The two scopes fight every frame. The "winner" is whichever rebuilds
last in the cycle, which is brittle and tree-order-dependent.

Rules of thumb:

- Add `reclaim_focus: true` to the scope of the **screen / view that
  owns the modal flow**. That's the scope that should regain focus
  after dismiss.
- Don't add it to inactive sibling scopes. They'll either grab focus
  unexpectedly or fight the active one.
- If multiple views can open modals, model focus restoration
  explicitly (e.g. save the previously-focused node via a `FocusNode`
  binding and re-focus on dismiss) rather than relying on multiple
  reclaim_focus scopes.

### Concrete recipe for a custom modal

1. The modal *frame* widget wraps its content in
   `focus_scope(trap: true, auto_focus: true, reclaim_focus: true)`.
   (`reclaim_focus` covers the case where the body subtree swaps
   mid-flight, e.g. a loading view replaced by the real picker — the
   loading-side focusable disposes, focus would be lost without
   reclaim.)
2. The frame does **not** wrap the caller's body in `ui::focus(...)`.
   That would put the focusable *above* whatever `Shortcuts` /
   `Actions` the body installs, and events would bubble past them.
   The caller's body must bring its own focusable inside its own
   handler layers.
3. The *view that opens the modal* uses
   `focus_scope(..., reclaim_focus: true)` around its body so focus
   snaps back into the view's subtree after dismiss — without
   relying on the index-based fallback to land somewhere useful.

---

## 10. Worked example

The minimal logged-in shell in `tinear/tui/logged_in_screen.ard` ended
up at this structure after working through every item above:

```
ui::shortcuts(
  ui::actions(
    ui::focus_scope(
      ui::focus(screen(…)),
      trap: false,
      auto_focus: true,
    ),
    [
      // app-level intents we want to repurpose for this screen
      ui::action("vaxis.next-focus",     …),   // Tab       → next tab
      ui::action("vaxis.previous-focus", …),   // Shift+Tab → prev tab
      ui::action("vaxis.dismiss",        …),   // Esc       → close active issue tab
      // custom intents we bind in our own Shortcuts below
      ui::action("tabs.inbox",     …),
      ui::action("tabs.my_issues", …),
    ],
  ),
  [
    "1": "tabs.inbox",
    "2": "tabs.my_issues",
  ],
)
```

And the app shell (`main.ard`) avoids wrapping the stateful root in
`ui::focus(...)` — there is no outer focus widget. The only focus
target is the one inside the logged-in scope. `focus_scope` with
`auto_focus: true` pulls focus there when the screen mounts.

---

## Changelog

- 2025-XX: added §9 on focus restoration after unmount — the
  built-in `pendingFocusFallback` mechanism, when it suffices,
  why nested handler layers need `reclaim_focus`, and the failure
  mode of putting `reclaim_focus: true` on every scope. Updated
  `ui::focus_scope` signature in §6 to include the new
  `reclaim_focus` parameter.
- 2025-04: initial doc capturing the event/focus/intent model and the
  "inner `Shortcuts` can't override `Tab`" pitfall surfaced during the
  tinear logged-in shell rewrite.
