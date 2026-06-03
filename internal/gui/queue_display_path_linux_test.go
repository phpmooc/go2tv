//go:build linux && !android

package gui

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/unix"
)

func TestNewQueueItemUsesDocumentPortalHostPathForDisplay(t *testing.T) {
	portalPath := filepath.Join(t.TempDir(), "portal-name.mp4")
	if err := os.WriteFile(portalPath, nil, 0o600); err != nil {
		t.Fatalf("write portal file: %v", err)
	}

	hostPath := "/home/user/Videos/original-name.mp4"
	if err := unix.Setxattr(portalPath, documentPortalHostPathXattr, []byte(hostPath), 0); err != nil {
		if errors.Is(err, unix.ENOTSUP) {
			t.Skipf("filesystem does not support xattrs: %v", err)
		}
		t.Fatalf("set document portal xattr: %v", err)
	}

	screen := newTraversalTestScreen(t, "")
	item, ok := screen.newQueueItem(portalPath)
	if !ok {
		t.Fatal("expected queue item")
	}

	if item.Path != portalPath {
		t.Fatalf("access path changed: got %q want %q", item.Path, portalPath)
	}
	if item.DisplayPath != hostPath {
		t.Fatalf("display path: got %q want %q", item.DisplayPath, hostPath)
	}
	if item.BaseName != filepath.Base(hostPath) {
		t.Fatalf("base name: got %q want %q", item.BaseName, filepath.Base(hostPath))
	}
	if item.ParentFolder != filepath.Dir(hostPath) {
		t.Fatalf("parent folder: got %q want %q", item.ParentFolder, filepath.Dir(hostPath))
	}
}

func TestQueueDisplayPathFallsBackWithoutDocumentPortalXattr(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ordinary.mp4")

	if got := queueDisplayPath(path); got != path {
		t.Fatalf("display path: got %q want %q", got, path)
	}
}
