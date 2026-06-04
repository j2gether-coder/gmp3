// ui/common.go
package ui

import (
	"strings"

	"fyne.io/fyne/v2/widget"
)

// MessageBox
func AppendLog(box *widget.Entry, text string) {
	if box == nil {
		return
	}

	// 자동 줄 바꿈 처리
	if !strings.HasSuffix(text, "\n") {
		text += "\n"
	}

	box.SetText(box.Text + text)
	//box.CursorRow = len(box.Text)
	box.CursorRow = len(strings.Split(box.Text, "\n")) - 1

	box.Refresh()
}
