package ffi

import (
	"git.sr.ht/~rockorager/vaxis/ui"
)

// ─── State / context plumbing ─────────────────────────────────────────

// UiEventContext is the vaxis/ui event context passed to callbacks.
type UiEventContext = ui.EventContext

// UiStateContext holds the typed state value for a stateful widget.
type UiStateContext struct {
	Value          any
	markNeedsBuild func()
}

func UiStateValue[T any](ctx *UiStateContext) T {
	return ctx.Value.(T)
}

// ─── Intent (string-backed) ────────────────────────────────────────────

type uiStringIntent string

func (i uiStringIntent) IntentType() ui.IntentType { return ui.IntentType(i) }

// ─── Layout widgets ───────────────────────────────────────────────────

func UiRow(children []ui.Widget) ui.Widget {
	return ui.Row(children...)
}

func UiColumn(children []ui.Widget) ui.Widget {
	return ui.Column(children...)
}

func UiCenter(child ui.Widget) ui.Widget {
	return ui.Center(child)
}

func UiPadding(all int, child ui.Widget) ui.Widget {
	return ui.Padding(ui.All(all), child)
}

func UiExpanded(child ui.Widget) ui.Widget {
	return ui.Expanded(child)
}

func UiSizedBox(width, height int, child ui.Widget) ui.Widget {
	return ui.SizedBox{Width: width, Height: height, Child: child}
}

// ─── Basic widgets ────────────────────────────────────────────────────

func UiText(value string) ui.Widget {
	return ui.Text{Value: value}
}

func UiButton(label string, onPressed func(ui.EventContext)) ui.Widget {
	return ui.Button{Label: label, OnPressed: onPressed}
}

// ─── Stateful widget ──────────────────────────────────────────────────

type uiStateful[T any] struct {
	Key   string
	Init  func(ui.BuildContext, *UiStateContext) T
	Build func(ui.BuildContext, *UiStateContext) ui.Widget
}

func (w uiStateful[T]) WidgetKey() ui.KeyValue { return ui.KeyValue(w.Key) }

func (w uiStateful[T]) CreateState() ui.State {
	s := &uiStatefulState[T]{widget: w, state: &UiStateContext{}}
	return s
}

type uiStatefulState[T any] struct {
	ui.StateBase
	widget uiStateful[T]
	state  *UiStateContext
}

func (s *uiStatefulState[T]) Build(ctx ui.BuildContext) ui.Widget {
	// Wire up rebuild trigger once the element is attached.
	if s.state.markNeedsBuild == nil {
		s.state.markNeedsBuild = s.MarkNeedsBuild
	}
	if s.state.Value == nil {
		s.state.Value = s.widget.Init(ctx, s.state)
	}
	return s.widget.Build(ctx, s.state)
}

func UiStateSetValue[T any](ctx *UiStateContext, value T) {
	ctx.Value = value
	if ctx.markNeedsBuild != nil {
		ctx.markNeedsBuild()
	}
}

func UiStateful[T any](
	key string,
	init func(ui.BuildContext, *UiStateContext) T,
	build func(ui.BuildContext, *UiStateContext) ui.Widget,
) ui.Widget {
	return uiStateful[T]{Key: key, Init: init, Build: build}
}

// ─── Actions ──────────────────────────────────────────────────────────

type UiActionBinding struct {
	Name    string
	Handler func(ui.EventContext, string) int
}

func NewUiActionBinding(name string, handler func(ui.EventContext, string) int) UiActionBinding {
	return UiActionBinding{Name: name, Handler: handler}
}

func UiActions(child ui.Widget, bindings []UiActionBinding) ui.Widget {
	m := make(map[ui.IntentType]ui.ActionFunc)
	for _, b := range bindings {
		name := b.Name
		handler := b.Handler
		m[ui.IntentType(name)] = func(ctx ui.EventContext, intent ui.Intent) ui.EventResult {
			return ui.EventResult(handler(ctx, string(intent.IntentType())))
		}
	}
	return ui.Actions{Bindings: m, Child: child}
}

func UiShortcuts(child ui.Widget, bindings map[string]string) ui.Widget {
	m := make(ui.ShortcutMap)
	for key, intentName := range bindings {
		m[key] = uiStringIntent(intentName)
	}
	return ui.Shortcuts{Bindings: m, Child: child}
}

// ─── Focus ────────────────────────────────────────────────────────────

func UiFocus(child ui.Widget) ui.Widget {
	return ui.Focus(nil, child)
}

func UiFocusScope(child ui.Widget, trap, autoFocus bool) ui.Widget {
	return ui.FocusScope{Trap: trap, AutoFocus: autoFocus, Child: child}
}

// ─── Divider ──────────────────────────────────────────────────────────

func UiDivider() ui.Widget {
	return ui.Divider{}
}

// ─── Text input ───────────────────────────────────────────────────────

func UiTextField(value, placeholder string, minWidth int, obscure bool, onChanged func(ui.EventContext, string), onSubmitted func(ui.EventContext, string)) ui.Widget {
	return ui.TextField{
		Value:       value,
		Placeholder: placeholder,
		MinWidth:    minWidth,
		ObscureText: obscure,
		OnChanged:   onChanged,
		OnSubmitted: onSubmitted,
	}
}

// ─── Scroll ───────────────────────────────────────────────────────────

func UiScrollView(child ui.Widget) ui.Widget {
	return ui.ScrollView{Child: child}
}

// ─── App runner ───────────────────────────────────────────────────────

func UiRun(root ui.Widget) error {
	return ui.Run(root)
}

// ─── Event helpers ────────────────────────────────────────────────────

func UiQuit(ctx ui.EventContext) { ctx.Quit() }
