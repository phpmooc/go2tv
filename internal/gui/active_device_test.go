//go:build !(android || ios)

package gui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func TestUpdateActiveDeviceViewUsesActiveDevice(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()

	screen := &FyneScreen{
		State:             "Playing",
		selectedDevice:    devType{name: "Bedroom TV", addr: "http://bedroom", deviceType: "chromecast"},
		activeDevice:      devType{name: "Living Room TV", addr: "http://living-room", deviceType: "chromecast"},
		ActiveDeviceLabel: widget.NewLabel(""),
		ActiveDeviceCard:  widget.NewCard("Active Device", "", container.NewHBox(widget.NewIcon(theme.MediaPlayIcon()), widget.NewLabel(""))),
	}
	screen.ActiveDeviceCard.Hide()

	fyne.DoAndWait(func() {
		screen.updateActiveDeviceView()
	})

	if got := screen.ActiveDeviceLabel.Text; got != "Living Room TV" {
		t.Fatalf("active device label = %q, want %q", got, "Living Room TV")
	}
	if !screen.ActiveDeviceCard.Visible() {
		t.Fatal("expected active device card visible")
	}
}

func TestClearActiveDeviceHidesActiveDeviceCard(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()

	screen := &FyneScreen{
		State:             "Playing",
		activeDevice:      devType{name: "Living Room TV", addr: "http://living-room", deviceType: "chromecast"},
		ActiveDeviceLabel: widget.NewLabel(""),
		ActiveDeviceCard: widget.NewCard("Active Device", "",
			container.NewHBox(widget.NewIcon(theme.MediaPlayIcon()), widget.NewLabel(""))),
	}

	fyne.DoAndWait(func() {
		screen.updateActiveDeviceView()
	})

	screen.clearActiveDevice()
	fyne.DoAndWait(func() {})

	if got := screen.getActiveDevice(); got.addr != "" || got.name != "" {
		t.Fatalf("expected active device cleared, got %+v", got)
	}
	if screen.ActiveDeviceCard.Visible() {
		t.Fatal("expected active device card hidden after clear")
	}
}
