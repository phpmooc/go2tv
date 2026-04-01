//go:build android || ios

package gui

import "go2tv.app/go2tv/v2/internal/crashlog"

func crashPath(crash *crashlog.Session) string {
	if crash == nil {
		return ""
	}

	return crash.PreviousCrashPath()
}

func showPendingCrashPopup(*FyneScreen) {}
