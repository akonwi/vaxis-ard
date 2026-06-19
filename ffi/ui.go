package ffi

import (
	"time"

	"go.rockorager.dev/vaxis"
	"go.rockorager.dev/vaxis/ui"
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

func UiPadding(top, right, bottom, left int, child ui.Widget) ui.Widget {
	return ui.Padding(ui.Insets{Top: top, Right: right, Bottom: bottom, Left: left}, child)
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

func UiText(
	value string, softWrap bool, align, overflow, maxLines int,
	hasOnPressed bool, onPressed func(ui.EventContext), clickAffordance int,
) ui.Widget {
	t := ui.Text{
		Value:           value,
		SoftWrap:        softWrap,
		Align:           ui.TextAlign(align),
		Overflow:        ui.TextOverflow(overflow),
		MaxLines:        maxLines,
		ClickAffordance: ui.ClickAffordance(clickAffordance),
	}
	if hasOnPressed {
		t.OnPressed = onPressed
	}
	return t
}

func UiStyledText(
	value string, fg, bg, ulColor, ulStyle, attrs int,
	hasOnPressed bool, onPressed func(ui.EventContext), clickAffordance int,
) ui.Widget {
	t := ui.Text{
		Value:           value,
		Style:           decodeUiStyle(fg, bg, ulColor, ulStyle, attrs),
		ClickAffordance: ui.ClickAffordance(clickAffordance),
	}
	if hasOnPressed {
		t.OnPressed = onPressed
	}
	return t
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
	return &uiStatefulState[T]{state: &UiStateContext{}}
}

type uiStatefulState[T any] struct {
	ui.StateBase
	state *UiStateContext
}

// Build reads the CURRENT widget configuration from StateBase, not
// from a cached field. The parent can pass an updated widget (e.g.
// a closure capturing a new `active` flag) and we must run that new
// closure on rebuild, not the one captured at CreateState time.
// Caching the widget in our own struct field was the source of a
// real bug: stateful subtrees kept executing the closure they were
// born with even after the parent rebuilt them with a new one.
func (s *uiStatefulState[T]) Build(ctx ui.BuildContext) ui.Widget {
	w := s.StateBase.Widget().(uiStateful[T])
	// Wire up rebuild trigger and state base once the element is attached.
	if s.state.markNeedsBuild == nil {
		s.state.markNeedsBuild = s.MarkNeedsBuild
		s.state.sb = &s.StateBase
	}
	if s.state.Value == nil {
		s.state.Value = w.Init(ctx, s.state)
	}
	return w.Build(ctx, s.state)
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

func UiFocusScope(child ui.Widget, trap, autoFocus, reclaimFocus bool) ui.Widget {
	return ui.FocusScope{
		Trap:         trap,
		AutoFocus:    autoFocus,
		ReclaimFocus: reclaimFocus,
		Child:        child,
	}
}

// ─── Divider ──────────────────────────────────────────────────────────

func UiDivider(axis, fg, bg, attrs int) ui.Widget {
	return ui.Divider{
		Axis:  ui.Axis(axis),
		Style: decodeUiStyle(fg, bg, -1, 0, attrs),
	}
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

func UiDecoratedBox(
	fg, bg, ulColor, ulStyle, attrs int,
	borderTop, borderRight, borderBottom, borderLeft bool,
	borderFg, borderBg, borderAttrs int,
	child ui.Widget,
) ui.Widget {
	return ui.DecoratedBox(
		ui.Decoration{
			Style: decodeUiStyle(fg, bg, ulColor, ulStyle, attrs),
			Border: ui.Border{
				Top: borderTop, Right: borderRight, Bottom: borderBottom, Left: borderLeft,
				Style: decodeUiStyle(borderFg, borderBg, -1, 0, borderAttrs),
			},
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

func UiIndexedStack(index int, children []ui.Widget, x, y int) ui.Widget {
	return ui.IndexedStack{Index: index, Children: children, Alignment: ui.Alignment{X: x, Y: y}}
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
	Title         string
	Description   string
	Aliases       []string
	HasOnSelected bool
	OnSelected    func(ui.EventContext)
}

// UiCommandPalette wires each per-item OnSelected if set; the global
// OnSelected fires after the per-item one and receives the item title
// (so an Ard caller can do any extra bookkeeping in one place).
func UiCommandPalette(
	items []UiCommandPaletteItem,
	placeholder, emptyText string,
	width, maxVisibleRows int,
	hasOnDismiss bool, onDismiss func(ui.EventContext),
	hasOnSelected bool, onSelected func(ui.EventContext, string),
) ui.Widget {
	built := make([]ui.CommandPaletteItem, len(items))
	for i, item := range items {
		entry := ui.CommandPaletteItem{
			Title:       item.Title,
			Description: item.Description,
			Aliases:     item.Aliases,
		}
		if item.HasOnSelected {
			entry.OnSelected = item.OnSelected
		}
		built[i] = entry
	}
	palette := ui.CommandPalette{
		Items:          built,
		Placeholder:    placeholder,
		EmptyText:      emptyText,
		Width:          width,
		MaxVisibleRows: maxVisibleRows,
	}
	if hasOnDismiss {
		palette.OnDismiss = onDismiss
	}
	if hasOnSelected {
		palette.OnSelected = func(ctx ui.EventContext, item ui.CommandPaletteItem) {
			onSelected(ctx, item.Title)
		}
	}
	return palette
}

func UiMakeCmdItem(
	title, description string,
	aliases []string,
	hasOnSelected bool,
	onSelected func(ui.EventContext),
) UiCommandPaletteItem {
	return UiCommandPaletteItem{
		Title: title, Description: description, Aliases: aliases,
		HasOnSelected: hasOnSelected, OnSelected: onSelected,
	}
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

// UiNewAnimation creates an AnimationController owned by the stateful
// element behind `stateCtx`. Curve is indexed: 0 linear, 1 ease_in_out.
func UiNewAnimation(stateCtx any, durationMs, curve int) *ui.AnimationController {
	ctx, ok := stateCtx.(*UiStateContext)
	if !ok || ctx == nil || ctx.sb == nil {
		return nil
	}
	opts := ui.AnimationOptions{
		Duration: time.Duration(durationMs) * time.Millisecond,
	}
	switch curve {
	case 1:
		opts.Curve = ui.EaseInOut
	default:
		opts.Curve = ui.Linear
	}
	return ctx.sb.NewAnimation(opts)
}

func UiAnimForward(ctrl *ui.AnimationController) { if ctrl != nil { ctrl.Forward() } }
func UiAnimReset(ctrl *ui.AnimationController)   { if ctrl != nil { ctrl.Reset() } }
func UiAnimStop(ctrl *ui.AnimationController)    { if ctrl != nil { ctrl.Stop() } }
func UiAnimValue(ctrl *ui.AnimationController) float64 {
	if ctrl == nil { return 0 }
	return ctrl.Value()
}

// UiAnimStatus returns the controller's status as an int:
// 0 idle, 1 forward (running), 2 completed.
func UiAnimStatus(ctrl *ui.AnimationController) int {
	if ctrl == nil { return 0 }
	return int(ctrl.Status())
}

// UiFloatToInt is a workaround for Ard compiler <= v0.23.0: the checker
// accepts `Float.to_int()` but the AIR lowerer is missing the FloatToInt
// branch (fixed on main, post-v0.23.0). Drop this once a release with
// the fix ships.
func UiFloatToInt(f float64) int { return int(f) }

func UiModalBarrier(fg, bg, ulColor, ulStyle, attrs int, opacity int) ui.Widget {
	return ui.ModalBarrier{Color: decodeUiStyle(fg, bg, ulColor, ulStyle, attrs).Background, Opacity: uint8(opacity)}
}

// UiListTile uses Bool sentinels for the four nullable slots so vaxis
// can distinguish "not set" from "set". Mirrors the span / text_field
// pattern. Avoids pulling ardruntime.Maybe across the FFI.
func UiListTile(
	title ui.Widget,
	selected, disabled bool,
	hasLeading bool, leading ui.Widget,
	hasSubtitle bool, subtitle ui.Widget,
	hasTrailing bool, trailing ui.Widget,
	hasOnPressed bool, onPressed func(ui.EventContext),
) ui.Widget {
	tile := ui.ListTile{Title: title, Selected: selected, Disabled: disabled}
	if hasLeading {
		tile.Leading = leading
	}
	if hasSubtitle {
		tile.Subtitle = subtitle
	}
	if hasTrailing {
		tile.Trailing = trailing
	}
	if hasOnPressed {
		tile.OnPressed = onPressed
	}
	return tile
}

// UiProgressBar binds vaxis ProgressBar. fg/bg use -1 for "inherit
// theme default"; underline fields aren't exposed (not needed for a
// progress fill). gradientStart/gradientEnd use -1 for "no gradient";
// when both are set, vaxis interpolates the filled portion across them.
func UiProgressBar(
	value float64, width int,
	filledFg, filledBg, filledAttrs int,
	emptyFg, emptyBg, emptyAttrs int,
	gradientStart, gradientEnd int,
) ui.Widget {
	bar := ui.ProgressBar{
		Value:       value,
		Width:       width,
		FilledStyle: decodeUiStyle(filledFg, filledBg, -1, 0, filledAttrs),
		EmptyStyle:  decodeUiStyle(emptyFg, emptyBg, -1, 0, emptyAttrs),
	}
	if gradientStart >= 0 {
		bar.GradientStart = colorFromInt(gradientStart)
	}
	if gradientEnd >= 0 {
		bar.GradientEnd = colorFromInt(gradientEnd)
	}
	return bar
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

func UiRichText(spans []UiTextSpan, softWrap bool, overflow, maxLines int) ui.Widget {
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
	return ui.RichText{
		Spans:    s,
		SoftWrap: softWrap,
		Overflow: ui.TextOverflow(overflow),
		MaxLines: maxLines,
	}
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

func UiScrollView(child ui.Widget, axis int, hasController bool, controller *ui.ScrollController) ui.Widget {
	sv := ui.ScrollView{Axis: ui.ScrollAxis(axis), Child: child}
	if hasController {
		sv.Controller = controller
	}
	return sv
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

// UiScrollPaneMetrics flattens a ScrollMetrics into a fixed-shape int
// slice the Ard side decodes into a struct. Order:
//   [scroll_offset, max_scroll_offset,
//    viewport_width, viewport_height,
//    content_width, content_height]
//
// Returns zero values when the controller has not been attached to a
// mounted pane (e.g. queried before the first layout). Upstream
// ScrollPaneController.Metrics handles the not-attached case safely.
func UiScrollPaneMetrics(controller *ui.ScrollPaneController, axis int) []int {
	m := controller.Metrics(ui.ScrollAxis(axis))
	return []int{
		m.ScrollOffset, m.MaxScrollOffset,
		m.ViewportWidth, m.ViewportHeight,
		m.ContentWidth, m.ContentHeight,
	}
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

// UiScrollControllerMetrics flattens ScrollMetrics into a fixed-shape
// int slice (same order as UiScrollPaneMetrics). ScrollController is
// single-axis, so there's no axis parameter — metrics are for the
// scroll_view's configured axis.
func UiScrollControllerMetrics(controller *ui.ScrollController) []int {
	m := controller.Metrics()
	return []int{
		m.ScrollOffset, m.MaxScrollOffset,
		m.ViewportWidth, m.ViewportHeight,
		m.ContentWidth, m.ContentHeight,
	}
}

func UiSliverListController() *ui.SliverListController {
	return &ui.SliverListController{}
}

// UiSliverListControllerRevealIndex scrolls the controlled
// SliverListBuilder just enough to bring `index` into view. Returns
// false if the controller isn't attached or no scroll was needed.
func UiSliverListControllerRevealIndex(controller *ui.SliverListController, index int) bool {
	return controller.RevealIndex(index)
}

// UiSliverListControllerScrollToIndex scrolls to `index` aligning it
// per `align`: 0 = start, 1 = center, 2 = end.
func UiSliverListControllerScrollToIndex(controller *ui.SliverListController, index, align int) bool {
	return controller.ScrollToIndex(index, ui.ScrollAlign(align))
}

func UiCustomScrollView(controller *ui.ScrollController, slivers []ui.Widget, followOutput bool) ui.Widget {
	return ui.CustomScrollView{Controller: controller, Slivers: slivers, FollowOutput: followOutput}
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

// UiSliverListBuilder is the lazy/index-based sliver. ItemExtent and
// EstimatedItemExtent are mutually exclusive in practice: set one or the
// other (or neither, for a default). Passing 0 leaves the field at its
// vaxis zero value.
//
// `hasController` toggles whether `controller` is attached; when false
// we pass nil to upstream so the builder runs uncontrolled.
func UiSliverListBuilder(
	count, itemExtent, estimatedItemExtent, overscan int,
	builder func(ui.BuildContext, int) ui.Widget,
	hasController bool, controller *ui.SliverListController,
) ui.Widget {
	if !hasController {
		controller = nil
	}
	return ui.SliverListBuilder{
		Controller:          controller,
		Count:               count,
		ItemExtent:          itemExtent,
		EstimatedItemExtent: estimatedItemExtent,
		Overscan:            overscan,
		Builder:             builder,
	}
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

func UiRunWithThemeSet(root ui.Widget, ts UiThemeSet) {
	if err := ui.Run(root, ui.WithThemeSet(ts.Set)); err != nil {
		panic(err)
	}
}

// ─── Event helpers ────────────────────────────────────────────────────

func UiQuit(ctx ui.EventContext) { ctx.Quit() }
func UiNotify(ctx ui.EventContext, title, body string) { ctx.Notify(title, body) }
func UiSetTitle(ctx ui.EventContext, title string) { ctx.SetTitle(title) }

// UiToggleProfileOverlay flips the built-in vaxis profiling overlay
// (shows frame timings + state churn over the app). Bound to Alt+P in
// the upstream demo.
func UiToggleProfileOverlay(ctx ui.EventContext) { ctx.ToggleProfileOverlay() }
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

// UiThemeColors returns all 22 semantic colors as a flat int slice.
// Order: background, foreground, surface, surface_raised, surface_hovered,
// surface_pressed, primary, primary_text, primary_hovered, primary_pressed,
// accent, accent_text, success, success_text, warning, warning_text,
// danger, danger_text, muted_foreground, disabled_foreground, selection, border.
func UiThemeColors(t UiTheme) []int {
	return []int{
		int(t.Theme.Background),
		int(t.Theme.Foreground),
		int(t.Theme.Surface),
		int(t.Theme.SurfaceRaised),
		int(t.Theme.SurfaceHovered),
		int(t.Theme.SurfacePressed),
		int(t.Theme.Primary),
		int(t.Theme.PrimaryText),
		int(t.Theme.PrimaryHovered),
		int(t.Theme.PrimaryPressed),
		int(t.Theme.Accent),
		int(t.Theme.AccentText),
		int(t.Theme.Success),
		int(t.Theme.SuccessText),
		int(t.Theme.Warning),
		int(t.Theme.WarningText),
		int(t.Theme.Danger),
		int(t.Theme.DangerText),
		int(t.Theme.MutedForeground),
		int(t.Theme.DisabledForeground),
		int(t.Theme.Selection),
		int(t.Theme.Border),
	}
}

// UiThemeSetColors writes all 22 semantic colors from a flat int slice.
func UiThemeSetColors(t UiTheme, colors []int) UiTheme {
	set := func(dst *vaxis.Color, i int) { if i < len(colors) { *dst = vaxis.Color(colors[i]) } }
	set(&t.Theme.Background, 0)
	set(&t.Theme.Foreground, 1)
	set(&t.Theme.Surface, 2)
	set(&t.Theme.SurfaceRaised, 3)
	set(&t.Theme.SurfaceHovered, 4)
	set(&t.Theme.SurfacePressed, 5)
	set(&t.Theme.Primary, 6)
	set(&t.Theme.PrimaryText, 7)
	set(&t.Theme.PrimaryHovered, 8)
	set(&t.Theme.PrimaryPressed, 9)
	set(&t.Theme.Accent, 10)
	set(&t.Theme.AccentText, 11)
	set(&t.Theme.Success, 12)
	set(&t.Theme.SuccessText, 13)
	set(&t.Theme.Warning, 14)
	set(&t.Theme.WarningText, 15)
	set(&t.Theme.Danger, 16)
	set(&t.Theme.DangerText, 17)
	set(&t.Theme.MutedForeground, 18)
	set(&t.Theme.DisabledForeground, 19)
	set(&t.Theme.Selection, 20)
	set(&t.Theme.Border, 21)
	return t
}

func UiThemeModeGet(t UiTheme) int  { return int(t.Theme.Mode) }
func UiThemeModeSet(t UiTheme, mode int) UiTheme {
	t.Theme.Mode = ui.ThemeMode(mode)
	return t
}

// UiThemePaletteScale returns the 11 generated tones (Tone50 .. Tone950)
// for one color family of the theme's palette. The family argument is
// indexed: 0 neutral, 1 red, 2 green, 3 yellow, 4 blue, 5 magenta, 6 cyan.
func UiThemePaletteScale(t UiTheme, family int) []int {
	var scale ui.ColorScale
	switch family {
	case 0:
		scale = t.Theme.Palette.Neutral
	case 1:
		scale = t.Theme.Palette.Red
	case 2:
		scale = t.Theme.Palette.Green
	case 3:
		scale = t.Theme.Palette.Yellow
	case 4:
		scale = t.Theme.Palette.Blue
	case 5:
		scale = t.Theme.Palette.Magenta
	case 6:
		scale = t.Theme.Palette.Cyan
	default:
		return nil
	}
	return []int{
		int(scale.Tone50),
		int(scale.Tone100),
		int(scale.Tone200),
		int(scale.Tone300),
		int(scale.Tone400),
		int(scale.Tone500),
		int(scale.Tone600),
		int(scale.Tone700),
		int(scale.Tone800),
		int(scale.Tone900),
		int(scale.Tone950),
	}
}

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

// UiThemeSetMake constructs a ThemeSet from explicit light + dark
// themes. Lets callers replace either variant — e.g. passing the
// dark theme for both slots to force dark rendering regardless of
// ColorThemeUpdate events from the terminal.
func UiThemeSetMake(light UiTheme, dark UiTheme) UiThemeSet {
	return UiThemeSet{Set: ui.ThemeSet{Light: light.Theme, Dark: dark.Theme}}
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
