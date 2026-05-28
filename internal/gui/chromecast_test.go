//go:build !(android || ios)

package gui

import (
	"testing"

	"github.com/alexballas/refyne/v2"
	"github.com/alexballas/refyne/v2/data/binding"
	"github.com/alexballas/refyne/v2/test"
	"github.com/alexballas/refyne/v2/widget"
)

func TestNextChromecastActionIDAdvancesGeneration(t *testing.T) {
	screen := &FyneScreen{chromecastActionID: 21}

	actionID := screen.nextChromecastActionID()
	if actionID != 22 {
		t.Fatalf("unexpected action id: got %d want %d", actionID, 22)
	}

	if !screen.isChromecastActionCurrent(actionID) {
		t.Fatal("expected new action id to be current")
	}

	if screen.isChromecastActionCurrent(21) {
		t.Fatal("expected old action id to be stale")
	}
}

func TestStartAfreshPlayButtonInvalidatesChromecastAction(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()

	currentPos := binding.NewString()
	endPos := binding.NewString()
	screen := &FyneScreen{
		chromecastActionID: 21,
		PlayPause:          widget.NewButton("", nil),
		SlideBar:           &tappedSlider{Slider: widget.NewSlider(0, 100)},
		CurrentPos:         currentPos,
		EndPos:             endPos,
		State:              "Playing",
	}

	startAfreshPlayButton(screen)
	fyne.DoAndWait(func() {})

	if got := screen.chromecastActionID; got != 22 {
		t.Fatalf("unexpected action id: got %d want %d", got, 22)
	}

	if screen.isChromecastActionCurrent(21) {
		t.Fatal("expected previous Chromecast action to be stale")
	}

	if got := screen.getScreenState(); got != "Stopped" {
		t.Fatalf("unexpected screen state: got %q want %q", got, "Stopped")
	}
}
