//go:build !(android || ios)

package gui

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteDiagnosticsIncludesCrashAndDebugLogs(t *testing.T) {
	crashDir := t.TempDir()
	crashPath := filepath.Join(crashDir, "20260402-120000.crash")
	if err := os.WriteFile(crashPath, []byte("panic: boom\nstack"), 0o600); err != nil {
		t.Fatalf("WriteFile crash: %v", err)
	}

	s := &FyneScreen{
		version:          "test",
		Debug:            newDebugWriter(),
		PendingCrashPath: crashPath,
	}
	_, _ = s.Debug.Write([]byte("line one\n"))
	_, _ = s.Debug.Write([]byte("line two\n"))

	var buf bytes.Buffer
	if err := writeDiagnostics(&buf, s); err != nil {
		t.Fatalf("writeDiagnostics: %v", err)
	}

	out := buf.String()
	for _, want := range []string{"Go2TV Diagnostics", "=== Crash Report ===", "panic: boom", "=== Debug Log Ring ===", "line one", "line two"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in diagnostics:\n%s", want, out)
		}
	}
}
