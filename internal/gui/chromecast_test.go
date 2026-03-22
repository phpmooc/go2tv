//go:build !(android || ios)

package gui

import (
	"context"
	"testing"
)

func TestNormalizeChromecastWatcherContextFallsBackToBackground(t *testing.T) {
	if normalizeChromecastWatcherContext(nil) == nil {
		t.Fatal("expected fallback context")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if got := normalizeChromecastWatcherContext(ctx); got != ctx {
		t.Fatal("expected existing context to be preserved")
	}
}

func TestNextChromecastActionIDAdvancesTrackingGeneration(t *testing.T) {
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
