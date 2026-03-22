//go:build !(android || ios)

package gui

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
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
