package ffi

import (
	"context"
	"fmt"
	"time"
	"unicode"
	"unicode/utf8"

	"git.sr.ht/~rockorager/vaxis"
)

// ─── Lifecycle ────────────────────────────────────────────────────────────

func Open(title string) (*vaxis.Vaxis, error) {
	return OpenWith(title, 1) // default: disableKittyKeyboard=true
}

// OpenWith creates a vaxis instance with bitpacked options.
// opts bits: 0=disableKittyKeyboard 1=disableMouse 2=noSignals
// 3=enableSGRPixels 4-7=csiuBitMask
func OpenWith(title string, opts int) (*vaxis.Vaxis, error) {
	vx, err := vaxis.New(vaxis.Options{
		DisableKittyKeyboard: opts&1 != 0,
		DisableMouse:         opts&2 != 0,
		NoSignals:            opts&4 != 0,
		EnableSGRPixels:      opts&8 != 0,
		CSIuBitMask:          vaxis.CSIuBitMask((opts >> 4) & 0xF),
	})
	if err != nil {
		return nil, err
	}
	if title != "" {
		vx.SetTitle(title)
	}
	vx.HideCursor()
	drainStartupEvents(vx)
	return vx, nil
}

func Close(vx *vaxis.Vaxis) error {
	if vx == nil {
		return nil
	}
	vx.Close()
	return nil
}

func Suspend(vx *vaxis.Vaxis) error {
	if vx == nil {
		return nil
	}
	return vx.Suspend()
}

func Resume(vx *vaxis.Vaxis) error {
	if vx == nil {
		return nil
	}
	return vx.Resume()
}

// ─── Render ───────────────────────────────────────────────────────────────

func Render(vx *vaxis.Vaxis) error {
	if vx == nil {
		return nil
	}
	vx.Render()
	return nil
}

func Refresh(vx *vaxis.Vaxis) {
	if vx != nil {
		vx.Refresh()
	}
}

func Width(vx *vaxis.Vaxis) int {
	if vx == nil {
		return 80
	}
	w, _ := vx.Window().Size()
	return w
}

func Height(vx *vaxis.Vaxis) int {
	if vx == nil {
		return 24
	}
	_, h := vx.Window().Size()
	return h
}

// ─── Cell buffer ──────────────────────────────────────────────────────────

// SetCell writes a single cell into the screen buffer at absolute coordinates.
// width=0 lets vaxis measure the grapheme width.
// fg/bg/ulColor: -1 for default, 0-255 indexed, >255 packed RGB.
// ulStyle: 0=off 1=single 2=double 3=curly 4=dotted 5=dashed.
// attrs bitmask: bold=1 dim=2 italic=4 blink=8 reverse=16 invisible=32 strikethrough=64.
func SetCell(
	vx *vaxis.Vaxis,
	col int, row int,
	grapheme string, width int,
	fg int, bg int,
	ulColor int, ulStyle int,
	attrs int,
) {
	if vx == nil {
		return
	}
	style := decodeStyle(fg, bg, ulColor, ulStyle, attrs)
	if width <= 0 {
		width = 1
	}
	vx.Window().SetCell(col, row, vaxis.Cell{
		Character: vaxis.Character{Grapheme: grapheme, Width: width},
		Style:     style,
	})
}

// Write prints a string at absolute coordinates, using vaxis's grapheme
// segmentation and width measurement (uniseg).
func Write(
	vx *vaxis.Vaxis,
	col, row int,
	text string,
	fg, bg, ulColor, ulStyle, attrs int,
) {
	if vx == nil || text == "" {
		return
	}
	style := decodeStyle(fg, bg, ulColor, ulStyle, attrs)
	win := vx.Window()
	for _, ch := range vaxis.Characters(text) {
		win.SetCell(col, row, vaxis.Cell{
			Character: ch,
			Style:     style,
		})
		col += ch.Width
	}
}

// Clear fills the given region with default-style spaces, resetting graphics.
func Clear(vx *vaxis.Vaxis, col, row, width, height int) {
	if vx == nil {
		return
	}
	win := vx.Window()
	for r := 0; r < height; r++ {
		for c := 0; c < width; c++ {
			win.SetCell(col+c, row+r, vaxis.Cell{
				Character: vaxis.Character{Grapheme: " ", Width: 1},
				Style:     vaxis.Style{},
			})
		}
	}
}

// Fill writes the same cell (with style) across the given region.
func Fill(
	vx *vaxis.Vaxis,
	col, row, width, height int,
	grapheme string,
	fg int, bg int,
	ulColor int, ulStyle int,
	attrs int,
) {
	if vx == nil {
		return
	}
	if grapheme == "" {
		grapheme = " "
	}
	style := decodeStyle(fg, bg, ulColor, ulStyle, attrs)
	win := vx.Window()
	for r := 0; r < height; r++ {
		for c := 0; c < width; c++ {
			win.SetCell(col+c, row+r, vaxis.Cell{
				Character: vaxis.Character{Grapheme: grapheme, Width: 1},
				Style:     style,
			})
		}
	}
}

// ─── Cursor ───────────────────────────────────────────────────────────────

func ShowCursor(vx *vaxis.Vaxis, col, row int) {
	if vx != nil {
		vx.ShowCursor(col, row, vaxis.CursorBlock)
	}
}

func ShowCursorStyled(vx *vaxis.Vaxis, col, row, style int) {
	if vx != nil {
		vx.ShowCursor(col, row, vaxis.CursorStyle(style))
	}
}

func HideCursor(vx *vaxis.Vaxis) {
	if vx != nil {
		vx.HideCursor()
	}
}

// Print prints segments with wrapping, returning encoded (col + row*10000).
func Print(
	vx *vaxis.Vaxis,
	col, row, width, height int,
	texts []string,
	fgs, bgs, ulColors, ulStyles, attrss []int,
) int {
	if vx == nil {
		return 0
	}
	segs := buildSegments(texts, fgs, bgs, ulColors, ulStyles, attrss)
	win := vx.Window().New(col, row, width, height)
	c, r := win.Print(segs...)
	return c + r*10000
}

// Println prints a single line, truncating if wider than the window.
func Println(
	vx *vaxis.Vaxis,
	col, row, width int,
	texts []string,
	fgs, bgs, ulColors, ulStyles, attrss []int,
) {
	if vx == nil {
		return
	}
	segs := buildSegments(texts, fgs, bgs, ulColors, ulStyles, attrss)
	win := vx.Window().New(col, row, width, 1)
	win.Println(row, segs...)
}

// PrintTruncate prints a single line with ... truncation.
func PrintTruncate(
	vx *vaxis.Vaxis,
	col, row, width int,
	texts []string,
	fgs, bgs, ulColors, ulStyles, attrss []int,
) {
	if vx == nil {
		return
	}
	segs := buildSegments(texts, fgs, bgs, ulColors, ulStyles, attrss)
	win := vx.Window().New(col, row, width, 1)
	win.PrintTruncate(row, segs...)
}

// Wrap uses unicode line-break logic, returning encoded (col + row*10000).
func Wrap(
	vx *vaxis.Vaxis,
	col, row, width, height int,
	texts []string,
	fgs, bgs, ulColors, ulStyles, attrss []int,
) int {
	if vx == nil {
		return 0
	}
	segs := buildSegments(texts, fgs, bgs, ulColors, ulStyles, attrss)
	win := vx.Window().New(col, row, width, height)
	c, r := win.Wrap(segs...)
	return c + r*10000
}

func buildSegments(
	texts []string,
	fgs, bgs, ulColors, ulStyles, attrss []int,
) []vaxis.Segment {
	n := len(texts)
	if len(fgs) < n {
		n = len(fgs)
	}
	segs := make([]vaxis.Segment, n)
	for i := 0; i < n; i++ {
		segs[i] = vaxis.Segment{
			Text:  texts[i],
			Style: decodeStyle(fgs[i], bgs[i], ulColors[i], ulStyles[i], attrss[i]),
		}
	}
	return segs
}

// SetStyle changes the style at a position without modifying its text.
func SetStyle(
	vx *vaxis.Vaxis,
	col, row int,
	fg, bg, ulColor, ulStyle, attrs int,
) {
	if vx == nil {
		return
	}
	style := decodeStyle(fg, bg, ulColor, ulStyle, attrs)
	vx.Window().SetStyle(col, row, style)
}

// ─── Events ───────────────────────────────────────────────────────────────

// ReadEvent reads the next event from vaxis and encodes it as a string:
//
//	"key:<normalized>"  — key press (e.g. "key:r", "key:up", "key:backspace")
//	"resize:<cols>:<rows>" — terminal resize
//	"focus:in" / "focus:out" — focus change
//	"paste:start" / "paste:end" — bracketed paste
//	"theme:dark" / "theme:light" — color scheme change
//	"redraw"             — redraw request
//	"quit"               — application closing
func ReadEvent(vx *vaxis.Vaxis) (string, error) {
	if vx == nil {
		return "quit", nil
	}
	for ev := range vx.Events() {
		switch ev := ev.(type) {
		case vaxis.Key:
			var normalized string
			if key := appKeyFromVaxisKey(ev); key != "" {
				normalized = key
			} else {
				switch ev.String() {
				case "Ctrl+c", "Esc":
					return "quit", nil
				case "Up":
					normalized = "up"
				case "Down":
					normalized = "down"
				case "Left":
					normalized = "left"
				case "Right":
					normalized = "right"
				case "Enter", "Space":
					normalized = "select"
				case "BackSpace":
					normalized = "backspace"
				default:
					continue
				}
			}
			return fmt.Sprintf("key:%d:%d:%s", int(ev.Modifiers), int(ev.EventType), normalized), nil
		case vaxis.Resize:
			return fmt.Sprintf("resize:%d:%d", ev.Cols, ev.Rows), nil
		case vaxis.Redraw:
			return "redraw", nil
		case vaxis.QuitEvent:
			return "quit", nil
		case vaxis.FocusIn:
			return "focus:in", nil
		case vaxis.FocusOut:
			return "focus:out", nil
		case vaxis.PasteStartEvent:
			return "paste:start", nil
		case vaxis.PasteEndEvent:
			return "paste:end", nil
		case vaxis.ColorThemeUpdate:
			if ev.Mode == vaxis.DarkMode {
				return "theme:dark", nil
			}
			return "theme:light", nil
		case vaxis.Mouse:
			return fmt.Sprintf("mouse:%d:%d:%d:%d:%d",
				int(ev.Button), ev.Col, ev.Row,
				int(ev.Modifiers), int(ev.EventType)), nil
		}
	}
	return "quit", nil
}

func drainStartupEvents(vx *vaxis.Vaxis) {
	quiet := time.NewTimer(100 * time.Millisecond)
	defer quiet.Stop()
	for {
		select {
		case <-vx.Events():
			if !quiet.Stop() {
				<-quiet.C
			}
			quiet.Reset(100 * time.Millisecond)
		case <-quiet.C:
			return
		}
	}
}

func appKeyFromVaxisKey(key vaxis.Key) string {
	for _, candidate := range []rune{firstRune(key.Text), key.Keycode, key.ShiftedCode} {
		if candidate == 0 || candidate > unicode.MaxRune || !unicode.IsPrint(candidate) {
			continue
		}
		switch candidate {
		case 'q', 'Q':
			return "q"
		case 'r', 'R':
			return "r"
		case 'k', 'K':
			return "up"
		case 'j', 'J':
			return "down"
		case 'h', 'H':
			return "left"
		case 'l', 'L':
			return "right"
		case ' ':
			return "select"
		default:
			return string(candidate)
		}
	}
	return ""
}

func firstRune(text string) rune {
	for _, r := range text {
		return r
	}
	return 0
}

// ─── Capabilities ──────────────────────────────────────────────────────────

func CanRGB(vx *vaxis.Vaxis) bool           { return vx != nil && vx.CanRGB() }
func CanSixel(vx *vaxis.Vaxis) bool         { return vx != nil && vx.CanSixel() }
func CanKittyGraphics(vx *vaxis.Vaxis) bool { return vx != nil && vx.CanKittyGraphics() }
func CanUnicodeCore(vx *vaxis.Vaxis) bool   { return vx != nil && vx.CanUnicodeCore() }
func CanDisplayGraphics(vx *vaxis.Vaxis) bool {
	return vx != nil && vx.CanDisplayGraphics()
}

// ─── Terminal helpers ──────────────────────────────────────────────────────

func Bell(vx *vaxis.Vaxis) {
	if vx != nil {
		vx.Bell()
	}
}

func SetTitle(vx *vaxis.Vaxis, title string) {
	if vx != nil {
		vx.SetTitle(title)
	}
}

func Notify(vx *vaxis.Vaxis, title, body string) {
	if vx != nil {
		vx.Notify(title, body)
	}
}

func ClipboardPush(vx *vaxis.Vaxis, text string) {
	if vx != nil {
		vx.ClipboardPush(text)
	}
}

func RenderedWidth(vx *vaxis.Vaxis, text string) int {
	if vx == nil {
		return len([]rune(text))
	}
	return vx.RenderedWidth(text)
}

func TerminalID(vx *vaxis.Vaxis) string {
	if vx == nil {
		return ""
	}
	return vx.TerminalID()
}

func TextBackspace(text string) string {
	if text == "" {
		return ""
	}
	_, size := utf8.DecodeLastRuneInString(text)
	if size <= 0 {
		return ""
	}
	return text[:len(text)-size]
}

// ─── P1: Mouse shape, clipboard pop, queries ─────────────────────────────

func SetMouseShape(vx *vaxis.Vaxis, shape string) {
	if vx != nil {
		vx.SetMouseShape(vaxis.MouseShape(shape))
	}
}

func ClipboardPop(vx *vaxis.Vaxis) (string, error) {
	if vx == nil {
		return "", nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return vx.ClipboardPop(ctx)
}

func NotifyWorkingDirectory(vx *vaxis.Vaxis, path string) {
	if vx != nil {
		vx.NotifyWorkingDirectory(path)
	}
}

func SetAppID(vx *vaxis.Vaxis, id string) {
	if vx != nil {
		vx.SetAppID(id)
	}
}

func QueryColor(vx *vaxis.Vaxis, index int) int {
	if vx == nil {
		return 0
	}
	c := vaxis.IndexColor(uint8(index))
	result := vx.QueryColor(c)
	if params := result.Params(); len(params) == 3 {
		return int(params[0])<<16 | int(params[1])<<8 | int(params[2])
	}
	return 0
}

func QueryForeground(vx *vaxis.Vaxis) int {
	if vx == nil {
		return 0
	}
	result := vx.QueryForeground()
	if params := result.Params(); len(params) == 3 {
		return int(params[0])<<16 | int(params[1])<<8 | int(params[2])
	}
	return 0
}

func QueryBackground(vx *vaxis.Vaxis) int {
	if vx == nil {
		return 0
	}
	result := vx.QueryBackground()
	if params := result.Params(); len(params) == 3 {
		return int(params[0])<<16 | int(params[1])<<8 | int(params[2])
	}
	return 0
}

func CanReportColor(vx *vaxis.Vaxis) bool          { return vx != nil && vx.CanReportColor() }
func CanReportForegroundColor(vx *vaxis.Vaxis) bool { return vx != nil && vx.CanReportForegroundColor() }
func CanReportBackgroundColor(vx *vaxis.Vaxis) bool { return vx != nil && vx.CanReportBackgroundColor() }
func CanSetAppID(vx *vaxis.Vaxis) bool              { return vx != nil && vx.CanSetAppID() }
func CanExplicitWidth(vx *vaxis.Vaxis) bool         { return vx != nil && vx.CanExplicitWidth() }
func CanInBandResize(vx *vaxis.Vaxis) bool          { return vx != nil && vx.CanInBandResize() }

// ─── Style decoding (shared) ──────────────────────────────────────────────

// decodeStyle converts the FlatBuffers-style encoded parameters into a
// vaxis.Style.  See SetCell for the encoding descriptions.
func decodeStyle(fg, bg, ulColor, ulStyle, attrs int) vaxis.Style {
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

func colorFromInt(value int) vaxis.Color {
	if value >= 0 && value <= 255 {
		return vaxis.IndexColor(uint8(value))
	}
	return vaxis.HexColor(uint32(value))
}
