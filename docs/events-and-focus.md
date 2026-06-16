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

### `ui::focus_scope(child, trap, auto_focus)`

A *scope* around a region of the tree. Not itself a focus target;
needs `ui::focus(...)` widgets inside it.

```ard
ui::focus_scope(
  ui::focus(inner),
  trap: true,        // keep Tab traversal inside this subtree
  auto_focus: true,  // pull focus into the scope when it mounts
)
```

- `auto_focus: true` calls `focusFirstWithin(scope)` on mount,
  guaranteeing focus lands somewhere inside the scope even if focus
  state was weird coming in (e.g. previous screen had focus on a
  field that just unmounted).
- `trap: true` keeps `Tab` / `Shift+Tab` cycling inside the scope.
  Useful for modal dialogs and overlay surfaces.

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

---

## 9. Worked example

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

- 2025-04: initial doc capturing the event/focus/intent model and the
  "inner `Shortcuts` can't override `Tab`" pitfall surfaced during the
  tinear logged-in shell rewrite.
