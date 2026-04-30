//go:build !(android || ios)

package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/widget"
)

func showPendingCrashPopup(s *FyneScreen) {
	if s == nil || s.PendingCrashPath == "" || s.Current == nil {
		return
	}

	fyne.Do(func() {
		lbl := widget.NewLabel(lang.L("Previous run crashed") + "\n" + lang.L("Diagnostics may contain private file paths, URLs, IPs, or device names."))
		lbl.Wrapping = fyne.TextWrapWord
		cnf := dialog.NewCustomConfirm(
			lang.L("Crash report"),
			lang.L("Export"),
			lang.L("Dismiss"),
			lbl,
			func(export bool) {
				if export {
					showDiagnosticsSaveDialog(s)
				}
			},
			s.Current,
		)
		cnf.Show()
	})
}
