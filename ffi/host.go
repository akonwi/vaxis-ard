package ffi

import (
	"time"
	"unicode"
	"unicode/utf8"

	"git.sr.ht/~rockorager/vaxis"
)

// ─── Lifecycle ────────────────────────────────────────────────────────────

func Open(title string) (*vaxis.Vaxis, error) {
	vx, err := vaxis.New(vaxis.Options{DisableKittyKeyboard: true})
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

func HideCursor(vx *vaxis.Vaxis) {
	if vx != nil {
		vx.HideCursor()
	}
}

// ─── Events ───────────────────────────────────────────────────────────────

func ReadKey(vx *vaxis.Vaxis) (string, error) {
	if vx == nil {
		return "q", nil
	}
	for ev := range vx.Events() {
		switch ev := ev.(type) {
		case vaxis.Key:
			if ev.EventType != vaxis.EventPress {
				continue
			}
			if key := appKeyFromVaxisKey(ev); key != "" {
				return key, nil
			}
			switch ev.String() {
			case "Ctrl+c", "Esc":
				return "q", nil
			case "Up":
				return "up", nil
			case "Down":
				return "down", nil
			case "Left":
				return "left", nil
			case "Right":
				return "right", nil
			case "Enter", "Space":
				return "select", nil
			case "BackSpace":
				return "backspace", nil
			}
		case vaxis.Resize, vaxis.Redraw:
			return "redraw", nil
		case vaxis.QuitEvent:
			return "q", nil
		}
	}
	return "q", nil
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
