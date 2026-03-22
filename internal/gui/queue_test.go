//go:build !(android || ios)

package gui

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

func newTraversalTestScreen(t *testing.T, currentPath string) *FyneScreen {
	t.Helper()

	return &FyneScreen{
		mediafile:    currentPath,
		mediaFormats: []string{".mp4", ".mp3", ".jpg"},
		videoFormats: []string{".mp4"},
		audioFormats: []string{".mp3"},
		imageFormats: []string{".jpg"},
		State:        "Stopped",
	}
}

func TestGetAdjacentQueuedMediaSameTypeOnly(t *testing.T) {
	dir := t.TempDir()
	videoOne := filepath.Join(dir, "01.mp4")
	audioOne := filepath.Join(dir, "02.mp3")
	videoTwo := filepath.Join(dir, "03.mp4")

	screen := newTraversalTestScreen(t, videoOne)
	screen.SkinNextOnlySameTypes = true
	screen.SessionQueue = newSessionQueue([]QueueItem{
		{Path: videoOne, BaseName: filepath.Base(videoOne), ParentFolder: dir, MediaType: "video"},
		{Path: audioOne, BaseName: filepath.Base(audioOne), ParentFolder: dir, MediaType: "audio"},
		{Path: videoTwo, BaseName: filepath.Base(videoTwo), ParentFolder: dir, MediaType: "video"},
	}, 0)

	name, path, err := getNextMediaOrError(screen)
	if err != nil {
		t.Fatalf("getNextMediaOrError failed: %v", err)
	}
	if name != filepath.Base(videoTwo) {
		t.Fatalf("unexpected next name: got %q want %q", name, filepath.Base(videoTwo))
	}
	if path != videoTwo {
		t.Fatalf("unexpected next path: got %q want %q", path, videoTwo)
	}
}

func TestGetAdjacentQueuedMediaStopsAtEnd(t *testing.T) {
	dir := t.TempDir()
	videoOne := filepath.Join(dir, "01.mp4")
	videoTwo := filepath.Join(dir, "02.mp4")

	screen := newTraversalTestScreen(t, videoTwo)
	screen.SessionQueue = newSessionQueue([]QueueItem{
		{Path: videoOne, BaseName: filepath.Base(videoOne), ParentFolder: dir, MediaType: "video"},
		{Path: videoTwo, BaseName: filepath.Base(videoTwo), ParentFolder: dir, MediaType: "video"},
	}, 1)

	_, _, err := getNextMediaOrError(screen)
	if !errors.Is(err, errNoNextQueueMedia) {
		t.Fatalf("expected errNoNextQueueMedia, got %v", err)
	}
}

func TestGetPreviousMediaFromFolderNoWrap(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "01.mp4")
	second := filepath.Join(dir, "02.mp4")

	for _, path := range []string{first, second} {
		if err := os.WriteFile(path, []byte("test"), 0o600); err != nil {
			t.Fatalf("WriteFile failed: %v", err)
		}
	}

	screen := newTraversalTestScreen(t, first)
	_, _, err := getPreviousMediaOrError(screen)
	if !errors.Is(err, errNoPreviousFolderMedia) {
		t.Fatalf("expected errNoPreviousFolderMedia, got %v", err)
	}
}

func TestGetNextMediaFromFolderWraps(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "01.mp4")
	second := filepath.Join(dir, "02.mp4")

	for _, path := range []string{first, second} {
		if err := os.WriteFile(path, []byte("test"), 0o600); err != nil {
			t.Fatalf("WriteFile failed: %v", err)
		}
	}

	screen := newTraversalTestScreen(t, second)
	name, path, err := getNextMediaOrError(screen)
	if err != nil {
		t.Fatalf("getNextMediaOrError failed: %v", err)
	}
	if name != filepath.Base(first) {
		t.Fatalf("unexpected wrapped name: got %q want %q", name, filepath.Base(first))
	}
	if path != first {
		t.Fatalf("unexpected wrapped path: got %q want %q", path, first)
	}
}

func TestGetPreviousQueuedMediaSameTypeOnly(t *testing.T) {
	dir := t.TempDir()
	videoOne := filepath.Join(dir, "01.mp4")
	audioOne := filepath.Join(dir, "02.mp3")
	videoTwo := filepath.Join(dir, "03.mp4")

	screen := newTraversalTestScreen(t, videoTwo)
	screen.SkinNextOnlySameTypes = true
	screen.SessionQueue = newSessionQueue([]QueueItem{
		{Path: videoOne, BaseName: filepath.Base(videoOne), ParentFolder: dir, MediaType: "video"},
		{Path: audioOne, BaseName: filepath.Base(audioOne), ParentFolder: dir, MediaType: "audio"},
		{Path: videoTwo, BaseName: filepath.Base(videoTwo), ParentFolder: dir, MediaType: "video"},
	}, 2)

	name, path, err := getPreviousMediaOrError(screen)
	if err != nil {
		t.Fatalf("getPreviousMediaOrError failed: %v", err)
	}
	if name != filepath.Base(videoOne) {
		t.Fatalf("unexpected previous name: got %q want %q", name, filepath.Base(videoOne))
	}
	if path != videoOne {
		t.Fatalf("unexpected previous path: got %q want %q", path, videoOne)
	}
}

func TestClearCurrentMediaSelectionPreservesQueue(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()

	screen := &FyneScreen{
		State:              "Stopped",
		mediafile:          "/tmp/test.mp4",
		MediaText:          widget.NewEntry(),
		SelectInternalSubs: widget.NewSelect(nil, nil),
		PlayPause:          widget.NewButton("", nil),
		SessionQueue: newSessionQueue([]QueueItem{
			{Path: "/tmp/test.mp4", BaseName: "test.mp4", ParentFolder: "/tmp", MediaType: "video"},
		}, 0),
	}

	screen.MediaText.SetText("test.mp4")
	clearCurrentMediaSelection(screen)

	if screen.SessionQueue == nil {
		t.Fatalf("expected queue to remain after media clear")
	}
	if screen.mediafile != "" {
		t.Fatalf("expected mediafile to be cleared, got %q", screen.mediafile)
	}
}

func TestQueueInteractionsLocked(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()

	screen := &FyneScreen{
		ExternalMediaURL: widget.NewCheck("", nil),
		rtmpServerCheck:  widget.NewCheck("", nil),
	}

	if screen.queueInteractionsLocked() {
		t.Fatalf("expected queue to be unlocked initially")
	}

	screen.ExternalMediaURL.SetChecked(true)
	if !screen.queueInteractionsLocked() {
		t.Fatalf("expected queue lock for external media mode")
	}

	screen.ExternalMediaURL.SetChecked(false)
	screen.rtmpServerCheck.SetChecked(true)
	if !screen.queueInteractionsLocked() {
		t.Fatalf("expected queue lock for RTMP mode")
	}

	screen.rtmpServerCheck.SetChecked(false)
	screen.Screencast = true
	if !screen.queueInteractionsLocked() {
		t.Fatalf("expected queue lock for screencast mode")
	}
}

func TestDroppedMediaBlockedError(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()

	screen := &FyneScreen{
		ExternalMediaURL: widget.NewCheck("", nil),
		rtmpServerCheck:  widget.NewCheck("", nil),
	}

	if err := screen.droppedMediaBlockedError(); err != nil {
		t.Fatalf("unexpected drop block without live mode: %v", err)
	}

	screen.ExternalMediaURL.SetChecked(true)
	if err := screen.droppedMediaBlockedError(); err != nil {
		t.Fatalf("external URL mode should allow dropping files, got %v", err)
	}

	screen.ExternalMediaURL.SetChecked(false)
	screen.rtmpServerCheck.SetChecked(true)
	if err := screen.droppedMediaBlockedError(); err == nil {
		t.Fatalf("expected drop block while RTMP mode is active")
	}

	screen.rtmpServerCheck.SetChecked(false)
	screen.Screencast = true
	if err := screen.droppedMediaBlockedError(); err == nil {
		t.Fatalf("expected drop block while screencast mode is active")
	}
}
