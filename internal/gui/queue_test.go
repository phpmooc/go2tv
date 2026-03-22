//go:build !(android || ios)

package gui

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"
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
	if screen.SessionQueue.CurrentIndex != -1 {
		t.Fatalf("expected queue current index to be cleared, got %d", screen.SessionQueue.CurrentIndex)
	}
	if screen.mediafile != "" {
		t.Fatalf("expected mediafile to be cleared, got %q", screen.mediafile)
	}
}

func TestSelectMediaPathsSingleFileCreatesQueue(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()

	dir := t.TempDir()
	mediaPath := filepath.Join(dir, "single.mp4")
	screen := &FyneScreen{
		mediaFormats:       []string{".mp4"},
		videoFormats:       []string{".mp4"},
		MediaText:          widget.NewEntry(),
		SubsText:           widget.NewEntry(),
		SelectInternalSubs: widget.NewSelect(nil, nil),
		CustomSubsCheck:    widget.NewCheck("", nil),
		PlayPause:          widget.NewButton("", nil),
	}

	if err := selectMediaPaths(screen, []string{mediaPath}); err != nil {
		t.Fatalf("selectMediaPaths failed: %v", err)
	}
	fyne.DoAndWait(func() {})

	if screen.SessionQueue == nil {
		t.Fatalf("expected single media selection to create queue")
	}
	if len(screen.SessionQueue.Items) != 1 {
		t.Fatalf("expected single queue item, got %d", len(screen.SessionQueue.Items))
	}
	if screen.SessionQueue.CurrentIndex != 0 {
		t.Fatalf("expected current queue index 0, got %d", screen.SessionQueue.CurrentIndex)
	}
	if screen.mediafile != mediaPath {
		t.Fatalf("expected mediafile %q, got %q", mediaPath, screen.mediafile)
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

func TestActiveQueueIndexRequiresCurrentMedia(t *testing.T) {
	screen := &FyneScreen{}
	queue := newSessionQueue([]QueueItem{
		{Path: "/tmp/one.mp4", BaseName: "one.mp4", ParentFolder: "/tmp", MediaType: "video"},
		{Path: "/tmp/two.mp4", BaseName: "two.mp4", ParentFolder: "/tmp", MediaType: "video"},
	}, 1)

	if got := screen.activeQueueIndex(queue); got != -1 {
		t.Fatalf("expected no active queue item without media selection, got %d", got)
	}

	screen.mediafile = "/tmp/two.mp4"
	if got := screen.activeQueueIndex(queue); got != 1 {
		t.Fatalf("expected active queue index 1, got %d", got)
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

func TestQueueDropMode(t *testing.T) {
	screen := &FyneScreen{}

	if got := screen.queueDropMode(); got != droppedMediaModeReplace {
		t.Fatalf("expected replace mode for empty queue, got %d", got)
	}

	screen.SessionQueue = newSessionQueue([]QueueItem{
		{Path: "/tmp/one.mp4", BaseName: "one.mp4", ParentFolder: "/tmp", MediaType: "video"},
	}, 0)

	if got := screen.queueDropMode(); got != droppedMediaModeAppend {
		t.Fatalf("expected append mode for existing queue, got %d", got)
	}
}

func TestDroppedMediaBlockedErrorForAppendMode(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()

	screen := &FyneScreen{
		ExternalMediaURL: widget.NewCheck("", nil),
		rtmpServerCheck:  widget.NewCheck("", nil),
	}

	if err := screen.droppedMediaBlockedErrorForMode(droppedMediaModeAppend); err != nil {
		t.Fatalf("unexpected append drop block without live mode: %v", err)
	}

	screen.ExternalMediaURL.SetChecked(true)
	if err := screen.droppedMediaBlockedErrorForMode(droppedMediaModeAppend); err == nil {
		t.Fatalf("expected append drop block while external URL mode is active")
	}

	if err := screen.droppedMediaBlockedErrorForMode(droppedMediaModeReplace); err != nil {
		t.Fatalf("replace mode should still allow dropping files, got %v", err)
	}
}

func TestRemoveSelectedQueueItemClearsCurrentWhenLastItemRemoved(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()

	mediaPath := "/tmp/test.mp4"
	screen := &FyneScreen{
		mediafile:          mediaPath,
		queueSelectedIndex: 0,
		MediaText:          widget.NewEntry(),
		SelectInternalSubs: widget.NewSelect(nil, nil),
		CustomSubsCheck:    widget.NewCheck("", nil),
		PlayPause:          widget.NewButton("", nil),
		SessionQueue: newSessionQueue([]QueueItem{
			{Path: mediaPath, BaseName: "test.mp4", ParentFolder: "/tmp", MediaType: "video"},
		}, 0),
	}

	screen.removeSelectedQueueItem()

	if screen.SessionQueue != nil {
		t.Fatalf("expected queue to be cleared after removing last item")
	}
	if screen.mediafile != "" {
		t.Fatalf("expected media selection to be cleared, got %q", screen.mediafile)
	}
}

func TestClearSessionQueueActionClearsCurrentMedia(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()

	mediaPath := "/tmp/test.mp4"
	screen := &FyneScreen{
		mediafile:          mediaPath,
		MediaText:          widget.NewEntry(),
		SelectInternalSubs: widget.NewSelect(nil, nil),
		CustomSubsCheck:    widget.NewCheck("", nil),
		PlayPause:          widget.NewButton("", nil),
		SessionQueue: newSessionQueue([]QueueItem{
			{Path: mediaPath, BaseName: "test.mp4", ParentFolder: "/tmp", MediaType: "video"},
		}, 0),
	}

	screen.clearSessionQueueAction()

	if screen.SessionQueue != nil {
		t.Fatalf("expected queue to be cleared")
	}
	if screen.mediafile != "" {
		t.Fatalf("expected media selection to be cleared, got %q", screen.mediafile)
	}
}

func TestSingleItemQueueButtonRemainsRed(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()

	mediaPath := "/tmp/test.mp4"
	screen := &FyneScreen{
		mediafile:   mediaPath,
		QueueButton: widget.NewButton("", nil),
		SessionQueue: newSessionQueue([]QueueItem{
			{Path: mediaPath, BaseName: "test.mp4", ParentFolder: "/tmp", MediaType: "video"},
		}, 0),
	}

	screen.refreshQueueStateUI()
	fyne.DoAndWait(func() {})

	if screen.QueueButton.Importance != widget.DangerImportance {
		t.Fatalf("expected single-item queue button to stay red, got %v", screen.QueueButton.Importance)
	}
}

func TestMultiItemQueueButtonTurnsGreen(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()

	mediaPath := "/tmp/one.mp4"
	screen := &FyneScreen{
		mediafile:   mediaPath,
		QueueButton: widget.NewButton("", nil),
		SessionQueue: newSessionQueue([]QueueItem{
			{Path: mediaPath, BaseName: "one.mp4", ParentFolder: "/tmp", MediaType: "video"},
			{Path: "/tmp/two.mp4", BaseName: "two.mp4", ParentFolder: "/tmp", MediaType: "video"},
		}, 0),
	}

	screen.refreshQueueStateUI()
	fyne.DoAndWait(func() {})

	if screen.QueueButton.Importance != widget.SuccessImportance {
		t.Fatalf("expected multi-item queue button to turn green, got %v", screen.QueueButton.Importance)
	}
}
