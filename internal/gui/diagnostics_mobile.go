//go:build android || ios

package gui

import (
	"bytes"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/lang"
	"go2tv.app/go2tv/v2/internal/crashlog"
)

func crashPath(crash *crashlog.Session) string {
	if crash == nil {
		return ""
	}

	return crash.PreviousCrashPath()
}

func copyDiagnosticsToClipboard(s *FyneScreen) {
	if s == nil || s.Current == nil {
		return
	}

	var buf bytes.Buffer
	if err := writeDiagnostics(&buf, s); err != nil {
		dialog.ShowError(err, s.Current)
		return
	}

	if app := fyne.CurrentApp(); app != nil {
		app.Clipboard().SetContent(buf.String())
	}

	dialog.ShowInformation(lang.L("Diagnostics"), lang.L("Copied to clipboard"), s.Current)
}

func showPendingCrashPopup(*FyneScreen) {}
