package ffi

import (
	"time"
	"unicode"
	"unicode/utf8"

	"git.sr.ht/~rockorager/vaxis"
)

func New(title string) (*vaxis.Vaxis, error) {
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

func Close(term *vaxis.Vaxis) error {
	if term == nil {
		return nil
	}
	term.Close()
	return nil
}

func Clear(term *vaxis.Vaxis) {
	if term == nil {
		return
	}
	term.Window().Clear()
}

func Refresh(term *vaxis.Vaxis) {
	if term != nil {
		term.Refresh()
	}
}

func Bell(term *vaxis.Vaxis) {
	if term != nil {
		term.Bell()
	}
}

func SetTitle(term *vaxis.Vaxis, title string) {
	if term != nil {
		term.SetTitle(title)
	}
}

func HideCursor(term *vaxis.Vaxis) {
	if term != nil {
		term.HideCursor()
	}
}

func ShowCursor(term *vaxis.Vaxis, x int, y int) {
	if term != nil {
		term.ShowCursor(x, y, vaxis.CursorBlock)
	}
}

func Suspend(term *vaxis.Vaxis) error {
	if term == nil {
		return nil
	}
	return term.Suspend()
}

func Resume(term *vaxis.Vaxis) error {
	if term == nil {
		return nil
	}
	return term.Resume()
}

func TerminalID(term *vaxis.Vaxis) string {
	if term == nil {
		return ""
	}
	return term.TerminalID()
}

func RenderedWidth(term *vaxis.Vaxis, text string) int {
	if term == nil {
		return len([]rune(text))
	}
	return term.RenderedWidth(text)
}

func CanRGB(term *vaxis.Vaxis) bool {
	return term != nil && term.CanRGB()
}

func CanSixel(term *vaxis.Vaxis) bool {
	return term != nil && term.CanSixel()
}

func CanKittyGraphics(term *vaxis.Vaxis) bool {
	return term != nil && term.CanKittyGraphics()
}

func CanUnicodeCore(term *vaxis.Vaxis) bool {
	return term != nil && term.CanUnicodeCore()
}

func CanDisplayGraphics(term *vaxis.Vaxis) bool {
	return term != nil && term.CanDisplayGraphics()
}

func Notify(term *vaxis.Vaxis, title string, body string) {
	if term != nil {
		term.Notify(title, body)
	}
}

func ClipboardPush(term *vaxis.Vaxis, text string) {
	if term != nil {
		term.ClipboardPush(text)
	}
}

func DrawText(term *vaxis.Vaxis, x int, y int, text string) {
	if term == nil {
		return
	}
	WindowDrawText(term.Window(), x, y, text)
}

func DrawTextStyle(term *vaxis.Vaxis, x int, y int, text string, fg int, bg int, bold bool, dim bool, italic bool, underline bool, reverse bool) {
	if term == nil {
		return
	}
	WindowDrawTextStyle(term.Window(), x, y, text, fg, bg, bold, dim, italic, underline, reverse)
}

func Render(term *vaxis.Vaxis) error {
	if term == nil {
		return nil
	}
	term.Render()
	return nil
}

func Root(term *vaxis.Vaxis) vaxis.Window {
	if term == nil {
		return vaxis.Window{}
	}
	return term.Window()
}

func RootWindow(term *vaxis.Vaxis, x int, y int, width int, height int) vaxis.Window {
	return Subwindow(Root(term), x, y, width, height)
}

func Subwindow(win vaxis.Window, x int, y int, width int, height int) vaxis.Window {
	if width < 0 {
		width = 0
	}
	if height < 0 {
		height = 0
	}
	return win.New(x, y, width, height)
}

func WindowWidth(win vaxis.Window) int {
	width, _ := win.Size()
	return width
}

func WindowHeight(win vaxis.Window) int {
	_, height := win.Size()
	return height
}

func WindowClear(win vaxis.Window) {
	win.Clear()
}

func WindowDrawText(win vaxis.Window, x int, y int, text string) {
	width, height := win.Size()
	if x < 0 || y < 0 || x >= width || y >= height {
		return
	}
	win.New(x, y, width-x, 1).Print(vaxis.Segment{Text: text})
}

func WindowDrawTextStyle(win vaxis.Window, x int, y int, text string, fg int, bg int, bold bool, dim bool, italic bool, underline bool, reverse bool) {
	width, height := win.Size()
	if x < 0 || y < 0 || x >= width || y >= height {
		return
	}
	win.New(x, y, width-x, 1).Print(vaxis.Segment{Text: text, Style: makeStyle(fg, bg, bold, dim, italic, underline, reverse)})
}

func WindowFill(win vaxis.Window, text string, fg int, bg int, bold bool, dim bool, italic bool, underline bool, reverse bool) {
	ch := " "
	if text != "" {
		ch = firstCharacter(text)
	}
	win.Fill(vaxis.Cell{Character: vaxis.Character{Grapheme: ch, Width: 1}, Style: makeStyle(fg, bg, bold, dim, italic, underline, reverse)})
}

func WindowShowCursor(win vaxis.Window, x int, y int) {
	win.ShowCursor(x, y, vaxis.CursorBlock)
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

func makeStyle(fg int, bg int, bold bool, dim bool, italic bool, underline bool, reverse bool) vaxis.Style {
	style := vaxis.Style{}
	if fg >= 0 {
		style.Foreground = colorFromInt(fg)
	}
	if bg >= 0 {
		style.Background = colorFromInt(bg)
	}
	if bold {
		style.Attribute |= vaxis.AttrBold
	}
	if dim {
		style.Attribute |= vaxis.AttrDim
	}
	if italic {
		style.Attribute |= vaxis.AttrItalic
	}
	if reverse {
		style.Attribute |= vaxis.AttrReverse
	}
	if underline {
		style.UnderlineStyle = vaxis.UnderlineSingle
	}
	return style
}

func colorFromInt(value int) vaxis.Color {
	if value >= 0 && value <= 255 {
		return vaxis.IndexColor(uint8(value))
	}
	return vaxis.HexColor(uint32(value))
}

func firstCharacter(text string) string {
	for _, r := range text {
		if r == 0 || !unicode.IsPrint(r) {
			break
		}
		return string(r)
	}
	return " "
}

func ReadKey(term *vaxis.Vaxis) (string, error) {
	if term == nil {
		return "q", nil
	}
	for ev := range term.Events() {
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
	// Do not use BaseLayoutCode here: it describes the physical PC-101 key and
	// can turn a typed app command like "r" into a movement key on alternate
	// layouts/protocols. App commands should follow the produced text/keycode.
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
