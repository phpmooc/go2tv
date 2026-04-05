package gui

import "fyne.io/fyne/v2"

const chromecastStartupBarWorkaroundPref = "ChromecastStartupBarWorkaround"

func chromecastStartupBarWorkaroundEnabled() bool {
	app := fyne.CurrentApp()
	if app == nil {
		return true
	}

	return app.Preferences().BoolWithFallback(chromecastStartupBarWorkaroundPref, true)
}
