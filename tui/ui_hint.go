package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// formHint is dim text under a form's fields that uses the form's whole width.
// A plain text item starts in the field column, after the longest label, so a
// form with long labels squeezed its hint into a few words a line and showed
// only its last lines.
type formHint struct{ *tview.TextView }

// SetFormAttributes keeps the hint at the left edge of the form: no label
// column.
func (h formHint) SetFormAttributes(_ int, labelColor, bgColor, fieldTextColor, fieldBgColor tcell.Color) tview.FormItem {
	h.TextView.SetFormAttributes(0, labelColor, bgColor, fieldTextColor, fieldBgColor)
	return h
}

// addHint adds text, rows lines high, across the whole width of the form.
func addHint(f *tview.Form, text string, rows int) {
	tv := tview.NewTextView().SetDynamicColors(true).SetWordWrap(true).SetScrollable(false)
	tv.SetSize(rows, 0)
	tv.SetText(text)
	f.AddFormItem(formHint{tv})
}
