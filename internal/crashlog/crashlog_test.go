package crashlog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRotatePendingReportMovesNonEmptyFile(t *testing.T) {
	dir := t.TempDir()
	pendingPath := filepath.Join(dir, pendingFilename)
	if err := os.WriteFile(pendingPath, []byte("panic"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	got, err := rotatePendingReport(dir)
	if err != nil {
		t.Fatalf("rotatePendingReport: %v", err)
	}

	if got == "" {
		t.Fatal("expected rotated crash path")
	}

	if _, err := os.Stat(got); err != nil {
		t.Fatalf("Stat rotated file: %v", err)
	}

	if _, err := os.Stat(pendingPath); !os.IsNotExist(err) {
		t.Fatalf("pending file still exists: %v", err)
	}

	b, err := os.ReadFile(got)
	if err != nil {
		t.Fatalf("ReadFile rotated file: %v", err)
	}

	if string(b) != "panic" {
		t.Fatalf("rotated contents = %q", string(b))
	}
}

func TestRotatePendingReportDeletesEmptyFile(t *testing.T) {
	dir := t.TempDir()
	pendingPath := filepath.Join(dir, pendingFilename)
	if err := os.WriteFile(pendingPath, nil, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	got, err := rotatePendingReport(dir)
	if err != nil {
		t.Fatalf("rotatePendingReport: %v", err)
	}

	if got != "" {
		t.Fatalf("expected no rotated crash, got %q", got)
	}

	if _, err := os.Stat(pendingPath); !os.IsNotExist(err) {
		t.Fatalf("pending file still exists: %v", err)
	}
}

func TestTrimReportsKeepsNewestFive(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, time.April, 2, 12, 0, 0, 0, time.UTC)

	for i := range 7 {
		path := filepath.Join(dir, now.Add(time.Duration(i)*time.Second).Format("20060102-150405")+crashExt)
		if err := os.WriteFile(path, []byte(path), 0o600); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
	}

	if err := trimReports(dir, 5); err != nil {
		t.Fatalf("trimReports: %v", err)
	}

	reports, err := crashReports(dir)
	if err != nil {
		t.Fatalf("crashReports: %v", err)
	}

	if len(reports) != 5 {
		t.Fatalf("report count = %d", len(reports))
	}

	if strings.Contains(filepath.Base(reports[len(reports)-1]), "120001") {
		t.Fatalf("oldest report was not trimmed: %v", reports)
	}
}

func TestTrimReportsNoopWhenBelowKeep(t *testing.T) {
	dir := t.TempDir()
	for i := range 2 {
		path := filepath.Join(dir, time.Date(2026, time.April, 2, 12, 0, i, 0, time.UTC).Format("20060102-150405")+crashExt)
		if err := os.WriteFile(path, []byte(path), 0o600); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
	}

	if err := trimReports(dir, 5); err != nil {
		t.Fatalf("trimReports: %v", err)
	}

	reports, err := crashReports(dir)
	if err != nil {
		t.Fatalf("crashReports: %v", err)
	}

	if len(reports) != 2 {
		t.Fatalf("report count = %d", len(reports))
	}
}

func TestCloseCleanIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	pendingPath := filepath.Join(dir, pendingFilename)
	if err := os.WriteFile(pendingPath, nil, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	session := &Session{pendingPath: pendingPath}
	if err := session.CloseClean(); err != nil {
		t.Fatalf("first CloseClean: %v", err)
	}

	if err := session.CloseClean(); err != nil {
		t.Fatalf("second CloseClean: %v", err)
	}

	if _, err := os.Stat(pendingPath); !os.IsNotExist(err) {
		t.Fatalf("pending file still exists: %v", err)
	}
}

func TestLatestCrashPathReturnsNewestReport(t *testing.T) {
	dir := t.TempDir()
	older := filepath.Join(dir, "20260402-120000.crash")
	newer := filepath.Join(dir, "20260402-120001.crash")
	if err := os.WriteFile(older, []byte("old"), 0o600); err != nil {
		t.Fatalf("WriteFile older: %v", err)
	}
	if err := os.WriteFile(newer, []byte("new"), 0o600); err != nil {
		t.Fatalf("WriteFile newer: %v", err)
	}

	session := &Session{dir: dir}
	if got := session.LatestCrashPath(); got != newer {
		t.Fatalf("LatestCrashPath = %q, want %q", got, newer)
	}
}
