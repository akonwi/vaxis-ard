package ffi

import (
	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/ui"
)

// ─── State / context plumbing ─────────────────────────────────────────

// UiEventContext is the vaxis/ui event context passed to callbacks.
type UiEventContext = ui.EventContext

// UiStateContext holds the typed state value for a stateful widget.
// `disposed` flips to true on unmount so background fibers calling
// rt.Dispatch become silent no-ops.
type UiStateContext struct {
	Value          any
	markNeedsBuild func()
	sb             *ui.StateBase
	disposed       bool
}

func UiStateValue[T any](ctx any) T {
	return ctx.(*UiStateContext).Value.(T)
}

// ─── Intent (string-backed) ────────────────────────────────────────────

type uiStringIntent string

func (i uiStringIntent) IntentType() ui.IntentType { return ui.IntentType(i) }

// ─── Layout widgets ───────────────────────────────────────────────────

func UiFlex(axis, mainAxisSize, mainAxisAlignment, crossAxisAlignment int, children []ui.Widget) ui.Widget {
	return ui.Flex{
		Axis:              ui.Axis(axis),
		MainAxisSize:      ui.MainAxisSize(mainAxisSize),
		MainAxisAlignment: ui.MainAxisAlignment(mainAxisAlignment),
		CrossAxisAlignment: ui.CrossAxisAlignment(crossAxisAlignment),
		Children:          children,
	}
}

func UiPositioned(left, top int, child ui.Widget) ui.Widget {
	return ui.Positioned{Left: left, Top: top, Child: child}
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

func UiFlexible(flex int, child ui.Widget) ui.Widget {
	return ui.FlexibleWidget{Flex: flex, Fit: ui.FlexFitLoose, Child: child}
}

func UiSizedBox(width, height int, child ui.Widget) ui.Widget {
	return ui.SizedBox{Width: width, Height: height, Child: child}
}

// ─── Basic widgets ────────────────────────────────────────────────────

func UiText(value string, softWrap bool, align, overflow, maxLines int) ui.Widget {
	return ui.Text{
		Value:    value,
		SoftWrap: softWrap,
		Align:    ui.TextAlign(align),
		Overflow: ui.TextOverflow(overflow),
		MaxLines: maxLines,
	}
}

func UiStyledText(value string, fg, bg, ulColor, ulStyle, attrs int) ui.Widget {
	return ui.Text{Value: value, Style: decodeUiStyle(fg, bg, ulColor, ulStyle, attrs)}
}

func UiButton(label string, onPressed func(ui.EventContext)) ui.Widget {
	return ui.Button{Label: label, OnPressed: onPressed}
}

func UiCheckbox(checked bool, disabled bool, label string, onChanged func(ui.EventContext, bool)) ui.Widget {
	return ui.Checkbox{Checked: checked, Disabled: disabled, Label: label, OnChanged: onChanged}
}

func UiRadio(value, groupValue string, disabled bool, label string, onChanged func(ui.EventContext, string)) ui.Widget {
	return ui.Radio[string]{Value: value, GroupValue: groupValue, Disabled: disabled, Label: label, OnChanged: onChanged}
}

type UiSegmentedItem struct {
	Value    string
	Label    string
	Disabled bool
}

func UiSegmentedControl(value string, segments []UiSegmentedItem, disabled bool, onChanged func(ui.EventContext, string)) ui.Widget {
	items := make([]ui.SegmentedItem[string], len(segments))
	for i, s := range segments {
		items[i] = ui.SegmentedItem[string]{Value: s.Value, Label: s.Label, Disabled: s.Disabled}
	}
	return ui.SegmentedControl[string]{Value: value, Segments: items, Disabled: disabled, OnChanged: onChanged}
}

func UiMakeSegmentedItem(value, label string, disabled bool) UiSegmentedItem {
	return UiSegmentedItem{Value: value, Label: label, Disabled: disabled}
}

// ─── Stateful widget ──────────────────────────────────────────────────

type uiStateful[T any] struct {
	Key   string
	Init  func(ui.BuildContext, any) T
	Build func(ui.BuildContext, any) ui.Widget
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
	// Wire up rebuild trigger and state base once the element is attached.
	if s.state.markNeedsBuild == nil {
		s.state.markNeedsBuild = s.MarkNeedsBuild
		s.state.sb = &s.StateBase
	}
	if s.state.Value == nil {
		s.state.Value = s.widget.Init(ctx, s.state)
	}
	return s.widget.Build(ctx, s.state)
}

func (s *uiStatefulState[T]) Dispose() {
	if s.state != nil {
		s.state.disposed = true
	}
}

func UiStateSetValue[T any](ctx any, value T) {
	c := ctx.(*UiStateContext)
	c.Value = value
	if c.markNeedsBuild != nil {
		c.markNeedsBuild()
	}
}

func UiStateful[T any](
	key string,
	init func(ui.BuildContext, any) T,
	build func(ui.BuildContext, any) ui.Widget,
) ui.Widget {
	return uiStateful[T]{Key: key, Init: init, Build: build}
}

// ─── Actions ──────────────────────────────────────────────────────────

type UiActionBinding struct {
	Name    string
	Handler func(ui.EventContext, string) int
}

func NewUiActionBinding(name string, handler func(ui.EventContext, string) int) any {
	return UiActionBinding{Name: name, Handler: handler}
}

func UiActions(child ui.Widget, bindings []any) ui.Widget {
	m := make(map[ui.IntentType]ui.ActionFunc)
	for _, b := range bindings {
		ab := b.(UiActionBinding); name := ab.Name
		handler := ab.Handler
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

func UiTextField(
	value, placeholder string,
	minWidth int,
	obscure bool,
	onChanged func(ui.EventContext, string),
	hasOnSubmitted bool,
	onSubmitted func(ui.EventContext, string),
) ui.Widget {
	f := ui.TextField{
		Value:       value,
		Placeholder: placeholder,
		MinWidth:    minWidth,
		ObscureText: obscure,
		OnChanged:   onChanged,
	}
	if hasOnSubmitted {
		f.OnSubmitted = onSubmitted
	}
	return f
}

func UiTextArea(value, placeholder string, minWidth, minHeight int, softWrap bool, onChanged func(ui.EventContext, string)) ui.Widget {
	return ui.TextArea{
		Value:       value,
		Placeholder: placeholder,
		MinWidth:    minWidth,
		MinHeight:   minHeight,
		SoftWrap:    softWrap,
		OnChanged:   onChanged,
	}
}

// ─── DecoratedBox ─────────────────────────────────────────────────────

func UiDecoratedBox(fg, bg, ulColor, ulStyle, attrs int, borderTop, borderRight, borderBottom, borderLeft bool, child ui.Widget) ui.Widget {
	return ui.DecoratedBox(
		ui.Decoration{
			Style:  decodeUiStyle(fg, bg, ulColor, ulStyle, attrs),
			Border: ui.Border{Top: borderTop, Right: borderRight, Bottom: borderBottom, Left: borderLeft, Style: decodeUiStyle(fg, bg, ulColor, ulStyle, attrs)},
		},
		child,
	)
}

func UiAlign(child ui.Widget, x, y int) ui.Widget {
	return ui.Align{Child: child, Alignment: ui.Alignment{X: x, Y: y}}
}

func UiConstrainedBox(child ui.Widget, minW, maxW, minH, maxH int) ui.Widget {
	return ui.ConstrainedBox{Child: child, Constraints: ui.Constraints{MinWidth: minW, MaxWidth: maxW, MinHeight: minH, MaxHeight: maxH}}
}

func UiStack(children []ui.Widget, x, y int) ui.Widget {
	return ui.Stack{Children: children, Alignment: ui.Alignment{X: x, Y: y}}
}

func UiDialog(title string, child ui.Widget, actions []ui.Widget, width int, onDismiss func(ui.EventContext)) ui.Widget {
	return ui.Dialog{Title: title, Child: child, Actions: actions, Width: width, OnDismiss: onDismiss}
}

type UiOverlayEntry struct {
	Child     ui.Widget
	Modal     bool
	AlignX    int
	AlignY    int
}

func UiOverlay(child ui.Widget, entries []UiOverlayEntry) ui.Widget {
	s := make([]ui.OverlayEntry, len(entries))
	for i, e := range entries {
		s[i] = ui.OverlayEntry{
			Child:     e.Child,
			Modal:     e.Modal,
			Alignment: ui.Alignment{X: e.AlignX, Y: e.AlignY},
		}
	}
	return ui.Overlay{Child: child, Entries: s}
}

func UiMakeOverlayEntry(child ui.Widget, modal bool, alignX, alignY int) UiOverlayEntry {
	return UiOverlayEntry{Child: child, Modal: modal, AlignX: alignX, AlignY: alignY}
}

func UiSelectionArea(child ui.Widget) ui.Widget {
	return ui.SelectionArea{Child: child}
}

func UiSelectionContainer(child ui.Widget, disabled bool) ui.Widget {
	return ui.SelectionContainer{Child: child, Disabled: disabled}
}

type UiCommandPaletteItem struct {
	Title       string
	Description string
	Aliases     []string
}

func UiCommandPalette(items []UiCommandPaletteItem, placeholder string, emptyText string, width int, maxVisibleRows int, onDismiss func(ui.EventContext), onSelected func(ui.EventContext, string)) ui.Widget {
	built := make([]ui.CommandPaletteItem, len(items))
	for i, item := range items {
		built[i] = ui.CommandPaletteItem{
			Title:       item.Title,
			Description: item.Description,
			Aliases:     item.Aliases,
		}
	}
	return ui.CommandPalette{
		Items:          built,
		Placeholder:    placeholder,
		EmptyText:      emptyText,
		Width:          width,
		MaxVisibleRows: maxVisibleRows,
		OnDismiss:      onDismiss,
		OnSelected: func(ctx ui.EventContext, item ui.CommandPaletteItem) {
			onSelected(ctx, item.Title)
		},
	}
}

func UiMakeCmdItem(title, description string, aliases []string) UiCommandPaletteItem {
	return UiCommandPaletteItem{Title: title, Description: description, Aliases: aliases}
}

type UiTableColumn struct {
	Kind  int
	Value int
}

type UiTableRow struct {
	Children []ui.Widget
}

func UiTable(columns []UiTableColumn, columnGap, rowGap int, rows []UiTableRow) ui.Widget {
	cols := make([]ui.TableColumn, len(columns))
	for i, c := range columns {
		switch c.Kind {
		case 1:
			cols[i] = ui.FixedColumn(c.Value)
		case 2:
			cols[i] = ui.FlexColumn(c.Value)
		default:
			cols[i] = ui.IntrinsicColumn()
		}
	}
	tblRows := make([]ui.TableRow, len(rows))
	for i, r := range rows {
		tblRows[i] = ui.TableRow{Children: r.Children}
	}
	return ui.Table{Columns: cols, ColumnGap: columnGap, RowGap: rowGap, Rows: tblRows}
}

func UiTableColumnIntrinsic() UiTableColumn { return UiTableColumn{Kind: 0} }
func UiTableColumnFixed(width int) UiTableColumn  { return UiTableColumn{Kind: 1, Value: width} }
func UiTableColumnFlex(flex int) UiTableColumn    { return UiTableColumn{Kind: 2, Value: flex} }
func UiTableRowNew(children []ui.Widget) UiTableRow { return UiTableRow{Children: children} }

func UiCursor(col, row, shape int, hidden bool, child ui.Widget) ui.Widget {
	return ui.Cursor{Col: col, Row: row, Shape: ui.CursorStyle(shape), Hidden: hidden, Child: child}
}

// ─── Animation ────────────────────────────────────────────────────────

func UiNewAnimation(stateCtx any, durationMs int) *ui.AnimationController {
	// Animation requires StateBase reference — not wired yet for dependency use.
	return nil
}

func UiAnimForward(ctrl *ui.AnimationController)  { if ctrl != nil { ctrl.Forward() } }
func UiAnimReset(ctrl *ui.AnimationController)   { if ctrl != nil { ctrl.Reset() } }
func UiAnimStop(ctrl *ui.AnimationController)    { if ctrl != nil { ctrl.Stop() } }
func UiAnimValue(ctrl *ui.AnimationController) float64 {
	if ctrl == nil { return 0 }
	return ctrl.Value()
}

func UiModalBarrier(fg, bg, ulColor, ulStyle, attrs int, opacity int) ui.Widget {
	return ui.ModalBarrier{Color: decodeUiStyle(fg, bg, ulColor, ulStyle, attrs).Background, Opacity: uint8(opacity)}
}

func UiListTile(title ui.Widget, selected bool, disabled bool, onPressed func(ui.EventContext)) ui.Widget {
	return ui.ListTile{Title: title, Selected: selected, Disabled: disabled, OnPressed: onPressed}
}

func UiProgressBar(value float64, width int, filledFg, filledBg, filledUlColor, filledUlStyle, filledAttrs, emptyFg, emptyBg, emptyUlColor, emptyUlStyle, emptyAttrs int) ui.Widget {
	return ui.ProgressBar{
		Value:       value,
		Width:       width,
		FilledStyle: decodeUiStyle(filledFg, filledBg, filledUlColor, filledUlStyle, filledAttrs),
		EmptyStyle:  decodeUiStyle(emptyFg, emptyBg, emptyUlColor, emptyUlStyle, emptyAttrs),
	}
}

// UiTextSpan carries a single styled run of text across the FFI boundary.
// HasOnPressed gates whether OnPressed is wired (vs. left nil so vaxis
// doesn't apply its interactive-style affordance to non-clickable spans).
type UiTextSpan struct {
	Text                            string
	Fg, Bg, UlColor, UlStyle, Attrs int
	Hyperlink                       string
	HasOnPressed                    bool
	OnPressed                       func(ui.EventContext)
}

func UiRichText(spans []UiTextSpan, softWrap bool) ui.Widget {
	s := make([]ui.TextSpan, len(spans))
	for i, sp := range spans {
		style := decodeUiStyle(sp.Fg, sp.Bg, sp.UlColor, sp.UlStyle, sp.Attrs)
		if sp.Hyperlink != "" {
			style.Hyperlink = sp.Hyperlink
		}
		s[i] = ui.TextSpan{Text: sp.Text, Style: style}
		if sp.HasOnPressed {
			s[i].OnPressed = sp.OnPressed
		}
	}
	return ui.RichText{Spans: s, SoftWrap: softWrap}
}

func UiMakeSpan(
	text string,
	attrs, fg, bg, ulStyle int,
	hyperlink string,
	hasOnPressed bool,
	onPressed func(ui.EventContext),
) UiTextSpan {
	return UiTextSpan{
		Text:         text,
		Attrs:        attrs,
		Fg:           fg,
		Bg:           bg,
		UlStyle:      ulStyle,
		Hyperlink:    hyperlink,
		HasOnPressed: hasOnPressed,
		OnPressed:    onPressed,
	}
}

// ─── Scroll ───────────────────────────────────────────────────────────

func UiScrollView(child ui.Widget, axis int) ui.Widget {
	return ui.ScrollView{Axis: ui.ScrollAxis(axis), Child: child}
}

func UiScrollbar(child ui.Widget, axis int) ui.Widget {
	return ui.Scrollbar{Axis: ui.ScrollAxis(axis), Child: child}
}

func UiScrollPaneController() *ui.ScrollPaneController {
	return &ui.ScrollPaneController{}
}

func UiScrollPane(controller *ui.ScrollPaneController, child ui.Widget) ui.Widget {
	return ui.ScrollPane{Controller: controller, Child: child}
}

func UiScrollPaneScrollBy(controller *ui.ScrollPaneController, cols, rows int) bool {
	return controller.ScrollBy(cols, rows)
}

func UiScrollPaneScrollTo(controller *ui.ScrollPaneController, col, row int) bool {
	return controller.ScrollTo(col, row)
}

func UiScrollController() *ui.ScrollController {
	return &ui.ScrollController{}
}

func UiScrollControllerScrollByLines(controller *ui.ScrollController, lines int) bool {
	return controller.ScrollByLines(lines)
}

func UiScrollControllerScrollToOffset(controller *ui.ScrollController, row int) bool {
	return controller.ScrollToOffset(row)
}

func UiScrollControllerScrollToStart(controller *ui.ScrollController) bool {
	return controller.ScrollToStart()
}

func UiScrollControllerScrollToEnd(controller *ui.ScrollController) bool {
	return controller.ScrollToEnd()
}

func UiCustomScrollView(controller *ui.ScrollController, slivers []ui.Widget) ui.Widget {
	return ui.CustomScrollView{Controller: controller, Slivers: slivers}
}

func UiSliverToBox(child ui.Widget) ui.Widget {
	return ui.SliverToBox{Child: child}
}

func UiSliverPinnedHeader(child ui.Widget) ui.Widget {
	return ui.SliverPinnedHeader{Child: child}
}

func UiSliverFillRemaining(child ui.Widget) ui.Widget {
	return ui.SliverFillRemaining{Child: child}
}

func UiSliverList(children []ui.Widget) ui.Widget {
	return ui.SliverList{Children: children}
}

// ─── App runner ───────────────────────────────────────────────────────

// All Ui*Run* functions return nothing. ui.Run only errors on backend
// setup failure (bad terminal, etc.) which is fatal at app start, so
// we panic to surface it rather than funnel it through Ard's Result.

func UiRun(root ui.Widget) {
	if err := ui.Run(root); err != nil {
		panic(err)
	}
}

func UiRunWithBaseColors(root ui.Widget, black, red, green, yellow, blue, magenta, cyan, white int) {
	err := ui.Run(root, ui.WithBaseColors(ui.BaseColors{
		Black:   colorFromInt(black),
		Red:     colorFromInt(red),
		Green:   colorFromInt(green),
		Yellow:  colorFromInt(yellow),
		Blue:    colorFromInt(blue),
		Magenta: colorFromInt(magenta),
		Cyan:    colorFromInt(cyan),
		White:   colorFromInt(white),
	}))
	if err != nil {
		panic(err)
	}
}

// ─── Event helpers ────────────────────────────────────────────────────

func UiQuit(ctx ui.EventContext) { ctx.Quit() }
func UiNotify(ctx ui.EventContext, title, body string) { ctx.Notify(title, body) }
func UiSetTitle(ctx ui.EventContext, title string) { ctx.SetTitle(title) }
func UiCopy(ctx ui.EventContext, text string) { ctx.Copy(text) }

// ─── Runtime dispatch ────────────────────────────────────────────────

func UiBuildContextRuntime(ctx ui.BuildContext) ui.Runtime {
	return ctx.Runtime()
}

// UiRuntimeDispatch schedules `callback` on the UI runtime, passing the
// typed state context back to the Ard side. Dispatches after the state's
// element has been unmounted are silent no-ops.
//
// `state` is the opaque StateHandle (`any`) from Ard; we type-assert it
// to *UiStateContext on entry, matching UiStateValue / UiStateSetValue.
func UiRuntimeDispatch(rt ui.Runtime, state any, callback func(any)) {
	if rt == nil || callback == nil || state == nil {
		return
	}
	ctx, ok := state.(*UiStateContext)
	if !ok || ctx == nil {
		return
	}
	rt.Dispatch(func() {
		if ctx.disposed {
			return
		}
		callback(ctx)
	})
}

// ─── Theme system ────────────────────────────────────────────────────

type UiTheme struct {
	Theme ui.Theme
}

func UiThemeDefault() UiTheme {
	return UiTheme{Theme: ui.DefaultTheme()}
}

func UiMustHaveTheme(ctx ui.BuildContext) UiTheme {
	return UiTheme{Theme: ui.MustDepend[ui.Theme](ctx)}
}

func UiProviderTheme(t UiTheme, child ui.Widget) ui.Widget {
	return ui.Provider[ui.Theme]{Value: t.Theme, Child: child}
}

func UiRunWithTheme(root ui.Widget, theme UiTheme) {
	if err := ui.Run(root, ui.WithTheme(theme.Theme)); err != nil {
		panic(err)
	}
}

// ─── Theme field setters ─────────────────────────────────────────────

func UiThemeWithBackground(t UiTheme, color int) UiTheme {
	t.Theme.Background = vaxis.Color(color)
	return t
}
func UiThemeWithForeground(t UiTheme, color int) UiTheme {
	t.Theme.Foreground = vaxis.Color(color)
	return t
}
func UiThemeWithSurface(t UiTheme, color int) UiTheme {
	t.Theme.Surface = vaxis.Color(color)
	return t
}
func UiThemeWithSurfaceRaised(t UiTheme, color int) UiTheme {
	t.Theme.SurfaceRaised = vaxis.Color(color)
	return t
}
func UiThemeWithSurfaceHovered(t UiTheme, color int) UiTheme {
	t.Theme.SurfaceHovered = vaxis.Color(color)
	return t
}
func UiThemeWithSurfacePressed(t UiTheme, color int) UiTheme {
	t.Theme.SurfacePressed = vaxis.Color(color)
	return t
}
func UiThemeWithPrimary(t UiTheme, color int) UiTheme {
	t.Theme.Primary = vaxis.Color(color)
	return t
}
func UiThemeWithPrimaryText(t UiTheme, color int) UiTheme {
	t.Theme.PrimaryText = vaxis.Color(color)
	return t
}
func UiThemeWithPrimaryHovered(t UiTheme, color int) UiTheme {
	t.Theme.PrimaryHovered = vaxis.Color(color)
	return t
}
func UiThemeWithPrimaryPressed(t UiTheme, color int) UiTheme {
	t.Theme.PrimaryPressed = vaxis.Color(color)
	return t
}
func UiThemeWithAccent(t UiTheme, color int) UiTheme {
	t.Theme.Accent = vaxis.Color(color)
	return t
}
func UiThemeWithAccentText(t UiTheme, color int) UiTheme {
	t.Theme.AccentText = vaxis.Color(color)
	return t
}
func UiThemeWithSuccess(t UiTheme, color int) UiTheme {
	t.Theme.Success = vaxis.Color(color)
	return t
}
func UiThemeWithSuccessText(t UiTheme, color int) UiTheme {
	t.Theme.SuccessText = vaxis.Color(color)
	return t
}
func UiThemeWithWarning(t UiTheme, color int) UiTheme {
	t.Theme.Warning = vaxis.Color(color)
	return t
}
func UiThemeWithWarningText(t UiTheme, color int) UiTheme {
	t.Theme.WarningText = vaxis.Color(color)
	return t
}
func UiThemeWithDanger(t UiTheme, color int) UiTheme {
	t.Theme.Danger = vaxis.Color(color)
	return t
}
func UiThemeWithDangerText(t UiTheme, color int) UiTheme {
	t.Theme.DangerText = vaxis.Color(color)
	return t
}
func UiThemeWithMutedForeground(t UiTheme, color int) UiTheme {
	t.Theme.MutedForeground = vaxis.Color(color)
	return t
}
func UiThemeWithDisabledForeground(t UiTheme, color int) UiTheme {
	t.Theme.DisabledForeground = vaxis.Color(color)
	return t
}
func UiThemeWithSelection(t UiTheme, color int) UiTheme {
	t.Theme.Selection = vaxis.Color(color)
	return t
}
func UiThemeWithBorder(t UiTheme, color int) UiTheme {
	t.Theme.Border = vaxis.Color(color)
	return t
}
func UiThemeWithMode(t UiTheme, mode int) UiTheme {
	t.Theme.Mode = ui.ThemeMode(mode)
	return t
}

// Theme field getters (all 22 semantic colors)
func UiThemeBackground(t UiTheme) int         { return int(t.Theme.Background) }
func UiThemeForeground(t UiTheme) int         { return int(t.Theme.Foreground) }
func UiThemeSurface(t UiTheme) int            { return int(t.Theme.Surface) }
func UiThemeSurfaceRaised(t UiTheme) int      { return int(t.Theme.SurfaceRaised) }
func UiThemeSurfaceHovered(t UiTheme) int     { return int(t.Theme.SurfaceHovered) }
func UiThemeSurfacePressed(t UiTheme) int     { return int(t.Theme.SurfacePressed) }
func UiThemePrimary(t UiTheme) int            { return int(t.Theme.Primary) }
func UiThemePrimaryText(t UiTheme) int        { return int(t.Theme.PrimaryText) }
func UiThemePrimaryHovered(t UiTheme) int     { return int(t.Theme.PrimaryHovered) }
func UiThemePrimaryPressed(t UiTheme) int     { return int(t.Theme.PrimaryPressed) }
func UiThemeAccent(t UiTheme) int             { return int(t.Theme.Accent) }
func UiThemeAccentText(t UiTheme) int         { return int(t.Theme.AccentText) }
func UiThemeSuccess(t UiTheme) int            { return int(t.Theme.Success) }
func UiThemeSuccessText(t UiTheme) int        { return int(t.Theme.SuccessText) }
func UiThemeWarning(t UiTheme) int            { return int(t.Theme.Warning) }
func UiThemeWarningText(t UiTheme) int        { return int(t.Theme.WarningText) }
func UiThemeDanger(t UiTheme) int             { return int(t.Theme.Danger) }
func UiThemeDangerText(t UiTheme) int         { return int(t.Theme.DangerText) }
func UiThemeMutedForeground(t UiTheme) int    { return int(t.Theme.MutedForeground) }
func UiThemeDisabledForeground(t UiTheme) int  { return int(t.Theme.DisabledForeground) }
func UiThemeSelection(t UiTheme) int          { return int(t.Theme.Selection) }
func UiThemeBorder(t UiTheme) int             { return int(t.Theme.Border) }
func UiThemeMode(t UiTheme) int              { return int(t.Theme.Mode) }

// ThemeSet helpers
func UiThemeLight() UiTheme {
	return UiTheme{Theme: ui.DefaultThemeSet().Light}
}
func UiThemeDark() UiTheme {
	return UiTheme{Theme: ui.DefaultThemeSet().Dark}
}

// UiThemeSet wraps a vaxis/ui ThemeSet (a light + dark Theme pair).
type UiThemeSet struct {
	Set ui.ThemeSet
}

func UiThemeSetDefault() UiThemeSet {
	return UiThemeSet{Set: ui.DefaultThemeSet()}
}

// UiRunWithThemeSet runs the app with both light + dark themes registered.
// vaxis swaps between them in response to terminal ColorThemeUpdate events.
func UiRunWithThemeSet(root ui.Widget, ts UiThemeSet) {
	if err := ui.Run(root, ui.WithThemeSet(ts.Set)); err != nil {
		panic(err)
	}
}

// ─── Animation helpers ───────────────────────────────────────────────

func decodeUiStyle(fg, bg, ulColor, ulStyle, attrs int) vaxis.Style {
	style := vaxis.Style{}
	if fg >= 0 {
		style.Foreground = colorFromInt(fg)
	}
	if bg >= 0 {
		style.Background = colorFromInt(bg)
	}
	if ulColor >= 0 {
		style.UnderlineColor = colorFromInt(ulColor)
	}
	style.UnderlineStyle = vaxis.UnderlineStyle(ulStyle)
	if attrs&1 != 0 {
		style.Attribute |= vaxis.AttrBold
	}
	if attrs&2 != 0 {
		style.Attribute |= vaxis.AttrDim
	}
	if attrs&4 != 0 {
		style.Attribute |= vaxis.AttrItalic
	}
	if attrs&8 != 0 {
		style.Attribute |= vaxis.AttrBlink
	}
	if attrs&16 != 0 {
		style.Attribute |= vaxis.AttrReverse
	}
	if attrs&32 != 0 {
		style.Attribute |= vaxis.AttrInvisible
	}
	if attrs&64 != 0 {
		style.Attribute |= vaxis.AttrStrikethrough
	}
	return style
}
