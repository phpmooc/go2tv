//go:build !(android || ios)

package gui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

func TestNormalizeChromecastWatcherContextUsesBackground(t *testing.T) {
	if normalizeChromecastWatcherContext(nil) == nil {
		t.Fatal("expected fallback context")
	}

	ctx := t.Context()

	if got := normalizeChromecastWatcherContext(ctx); got != ctx {
		t.Fatal("expected existing context to be preserved")
	}
}

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
