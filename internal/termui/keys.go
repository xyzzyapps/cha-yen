package termui

import (
	"unicode"

	"github.com/xyzzyapps/cha-yen/internal/termcore"
	"fyne.io/fyne/v2"
)

// mapKey translates a Fyne key event into a terminal key press event.
// ok is false when the key has no terminal equivalent.
func mapKey(k *fyne.KeyEvent, ctrl bool) (termcore.KeyPressEvent, bool) {
	var mod termcore.Mod
	if ctrl {
		mod |= termcore.ModCtrl
	}
	switch k.Name {
	case fyne.KeyReturn:
		return termcore.KeyPressEvent{Code: termcore.KeyEnter, Mod: mod}, true
	case fyne.KeyTab:
		return termcore.KeyPressEvent{Code: termcore.KeyTab, Mod: mod}, true
	case fyne.KeyEscape:
		return termcore.KeyPressEvent{Code: termcore.KeyEscape}, true
	case fyne.KeyBackspace:
		return termcore.KeyPressEvent{Code: termcore.KeyBackspace, Mod: mod}, true
	case fyne.KeyDelete:
		return termcore.KeyPressEvent{Code: termcore.KeyDelete, Mod: mod}, true
	case fyne.KeyUp:
		return termcore.KeyPressEvent{Code: termcore.KeyUp, Mod: mod}, true
	case fyne.KeyDown:
		return termcore.KeyPressEvent{Code: termcore.KeyDown, Mod: mod}, true
	case fyne.KeyLeft:
		return termcore.KeyPressEvent{Code: termcore.KeyLeft, Mod: mod}, true
	case fyne.KeyRight:
		return termcore.KeyPressEvent{Code: termcore.KeyRight, Mod: mod}, true
	case fyne.KeyHome:
		return termcore.KeyPressEvent{Code: termcore.KeyHome, Mod: mod}, true
	case fyne.KeyEnd:
		return termcore.KeyPressEvent{Code: termcore.KeyEnd, Mod: mod}, true
	case fyne.KeyPageUp:
		return termcore.KeyPressEvent{Code: termcore.KeyPgUp, Mod: mod}, true
	case fyne.KeyPageDown:
		return termcore.KeyPressEvent{Code: termcore.KeyPgDown, Mod: mod}, true
	}
	return termcore.KeyPressEvent{}, false
}

// mapRune translates a printable rune (soft keyboard or typing). Control
// characters are delivered as key codes; everything else as text.
func mapRune(r rune, ctrl bool) (termcore.KeyPressEvent, string, bool) {
	if ctrl && r != 0 {
		return termcore.KeyPressEvent{Code: unicode.ToLower(r), Mod: termcore.ModCtrl}, "", true
	}
	if r == ' ' {
		return termcore.KeyPressEvent{Code: termcore.KeySpace}, "", true
	}
	if r < 0x20 { // control char from a physical keyboard
		return termcore.KeyPressEvent{Code: r, Mod: termcore.ModCtrl}, "", true
	}
	return termcore.KeyPressEvent{}, string(r), true
}
