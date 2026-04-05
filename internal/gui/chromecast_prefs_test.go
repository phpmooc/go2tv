//go:build !(android || ios)

package gui

import (
	"testing"

	"fyne.io/fyne/v2/test"
)

func TestChromecastStartupBarWorkaroundEnabledDefaultsToTrue(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()

	if !chromecastStartupBarWorkaroundEnabled() {
		t.Fatal("expected Chromecast progress bar workaround to default to true")
	}
}

func TestChromecastStartupBarWorkaroundEnabledReadsPreference(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()

	app.Preferences().SetBool(chromecastStartupBarWorkaroundPref, false)

	if chromecastStartupBarWorkaroundEnabled() {
		t.Fatal("expected Chromecast progress bar workaround to follow stored preference")
	}
}
