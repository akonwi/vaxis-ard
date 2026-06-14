package ffi

import (
	"git.sr.ht/~rockorager/vaxis"
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

func UiSizedBox(width, height int, child ui.Widget) ui.Widget {
	return ui.SizedBox{Width: width, Height: height, Child: child}
}

// ─── Basic widgets ────────────────────────────────────────────────────

func UiText(value string) ui.Widget {
	return ui.Text{Value: value}
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

func UiConstrainedBox(minW, maxW, minH, maxH int, child ui.Widget) ui.Widget {
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

type UiTextSpan struct {
	Text           string
	Fg, Bg, UlColor, UlStyle, Attrs int
}

func UiRichText(spans []UiTextSpan, softWrap bool) ui.Widget {
	s := make([]ui.TextSpan, len(spans))
	for i, sp := range spans {
		s[i] = ui.TextSpan{
			Text:  sp.Text,
			Style: decodeUiStyle(sp.Fg, sp.Bg, sp.UlColor, sp.UlStyle, sp.Attrs),
		}
	}
	return ui.RichText{Spans: s, SoftWrap: softWrap}
}

func UiMakeSpan(text string, fg, bg, attrs int) UiTextSpan {
	return UiTextSpan{Text: text, Fg: fg, Bg: bg, Attrs: attrs}
}

// ─── Scroll ───────────────────────────────────────────────────────────

func UiScrollView(child ui.Widget) ui.Widget {
	return ui.ScrollView{Child: child}
}

func UiScrollbar(axis int, child ui.Widget) ui.Widget {
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

func UiRun(root ui.Widget) error {
	return ui.Run(root)
}

// ─── Event helpers ────────────────────────────────────────────────────

func UiQuit(ctx ui.EventContext) { ctx.Quit() }

// ─── Style decoding ───────────────────────────────────────────────────

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
