//go:build !(android || ios)

package gui

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type fakeResumePrefs struct {
	values map[string]string
}

type fakeDLNAResumeClient struct {
	transportInfo []string
	transportErr  error
	positionInfo  []string
	positionErr   error
	seekErr       error
}

func newFakeResumePrefs() *fakeResumePrefs {
	return &fakeResumePrefs{
		values: make(map[string]string),
	}
}

func (f *fakeResumePrefs) String(key string) string {
	return f.values[key]
}

func (f *fakeResumePrefs) SetString(key string, value string) {
	f.values[key] = value
}

func (f fakeDLNAResumeClient) GetTransportInfo() ([]string, error) {
	return f.transportInfo, f.transportErr
}

func (f fakeDLNAResumeClient) GetPositionInfo() ([]string, error) {
	return f.positionInfo, f.positionErr
}

func (f fakeDLNAResumeClient) SeekSoapCall(string) error {
	return f.seekErr
}

func TestResumeStoreRoundTrip(t *testing.T) {
	prefs := newFakeResumePrefs()
	store := newResumeStore(prefs)
	entry := resumeEntry{
		MediaPath:           "/tmp/example.mp4",
		Size:                123,
		MTimeUnixNano:       456,
		MediaKind:           "video",
		LastPositionSeconds: 42,
		UpdatedAtUnix:       time.Now().Unix(),
	}

	if err := store.save(entry); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	entries, err := store.load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	if entries[0] != entry {
		t.Fatalf("loaded entry mismatch: got %#v want %#v", entries[0], entry)
	}
}

func TestResumeStorePrunesToMaxEntries(t *testing.T) {
	prefs := newFakeResumePrefs()
	store := newResumeStore(prefs)
	entries := make([]resumeEntry, 0, resumeHistoryMaxEntries+5)

	for i := range resumeHistoryMaxEntries + 5 {
		entries = append(entries, resumeEntry{
			MediaPath:           filepath.ToSlash(filepath.Join("/tmp", "file-"+time.Unix(int64(i), 0).Format("150405")+".mp4")),
			Size:                int64(i),
			MTimeUnixNano:       int64(i),
			MediaKind:           "video",
			LastPositionSeconds: 20,
			UpdatedAtUnix:       int64(i),
		})
	}

	if err := store.saveEntries(entries); err != nil {
		t.Fatalf("saveEntries failed: %v", err)
	}

	loaded, err := store.load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if len(loaded) != resumeHistoryMaxEntries {
		t.Fatalf("expected %d entries, got %d", resumeHistoryMaxEntries, len(loaded))
	}

	if loaded[0].UpdatedAtUnix != int64(resumeHistoryMaxEntries+4) {
		t.Fatalf("expected newest entry first, got %d", loaded[0].UpdatedAtUnix)
	}
}

func TestResumeStoreReplacesMatchingEntry(t *testing.T) {
	prefs := newFakeResumePrefs()
	store := newResumeStore(prefs)
	first := resumeEntry{
		MediaPath:           "/tmp/example.mp4",
		Size:                123,
		MTimeUnixNano:       456,
		MediaKind:           "video",
		LastPositionSeconds: 20,
		UpdatedAtUnix:       1,
	}
	second := first
	second.LastPositionSeconds = 80
	second.UpdatedAtUnix = 2

	if err := store.save(first); err != nil {
		t.Fatalf("first save failed: %v", err)
	}
	if err := store.save(second); err != nil {
		t.Fatalf("second save failed: %v", err)
	}

	loaded, err := store.load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(loaded))
	}

	if loaded[0].LastPositionSeconds != 80 {
		t.Fatalf("expected replacement entry, got %d", loaded[0].LastPositionSeconds)
	}
}

func TestResumeStoreIgnoresCorruptJSON(t *testing.T) {
	prefs := newFakeResumePrefs()
	prefs.SetString(resumeHistoryPref, "{not-json")
	store := newResumeStore(prefs)

	loaded, err := store.load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if len(loaded) != 0 {
		t.Fatalf("expected empty entries, got %d", len(loaded))
	}
}

func TestResumeStoreSupportsLegacyArrayJSON(t *testing.T) {
	prefs := newFakeResumePrefs()
	store := newResumeStore(prefs)
	legacy := []resumeEntry{{
		MediaPath:           "/tmp/example.mp3",
		Size:                55,
		MTimeUnixNano:       77,
		MediaKind:           "audio",
		LastPositionSeconds: 33,
		UpdatedAtUnix:       44,
	}}
	encoded, err := json.Marshal(legacy)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	prefs.SetString(resumeHistoryPref, string(encoded))

	loaded, err := store.load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if len(loaded) != 1 || loaded[0].MediaPath != legacy[0].MediaPath {
		t.Fatalf("legacy load mismatch: %#v", loaded)
	}
}

func TestResumeStoreRemove(t *testing.T) {
	prefs := newFakeResumePrefs()
	store := newResumeStore(prefs)
	entry := resumeEntry{
		MediaPath:           "/tmp/example.mp4",
		Size:                1,
		MTimeUnixNano:       2,
		MediaKind:           "video",
		LastPositionSeconds: 12,
		UpdatedAtUnix:       3,
	}

	if err := store.save(entry); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	if err := store.remove(entry.identity()); err != nil {
		t.Fatalf("remove failed: %v", err)
	}

	loaded, err := store.load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if len(loaded) != 0 {
		t.Fatalf("expected empty store, got %d entries", len(loaded))
	}
}

func TestResumeEligibleForPlayback(t *testing.T) {
	tests := []struct {
		name       string
		external   bool
		screencast bool
		rtmp       bool
		mediaType  string
		wantKind   string
		wantOK     bool
	}{
		{name: "local_video", mediaType: "video/mp4", wantKind: "video", wantOK: true},
		{name: "local_audio", mediaType: "audio/flac", wantKind: "audio", wantOK: true},
		{name: "image", mediaType: "image/png", wantOK: false},
		{name: "external", external: true, mediaType: "video/mp4", wantOK: false},
		{name: "screencast", screencast: true, mediaType: "video/mp4", wantOK: false},
		{name: "rtmp", rtmp: true, mediaType: "video/mp4", wantOK: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotKind, gotOK := resumeEligibleForPlayback(tc.external, tc.screencast, tc.rtmp, tc.mediaType)
			if gotOK != tc.wantOK || gotKind != tc.wantKind {
				t.Fatalf("got (%q,%v) want (%q,%v)", gotKind, gotOK, tc.wantKind, tc.wantOK)
			}
		})
	}
}

func TestComputeResumeStart(t *testing.T) {
	tests := []struct {
		name         string
		existingSeek int
		storedResume int
		transcode    bool
		wantFFmpeg   int
		wantDirect   int
	}{
		{name: "manual_seek_wins", existingSeek: 77, storedResume: 30, transcode: true, wantFFmpeg: 77},
		{name: "transcode_resume", storedResume: 35, transcode: true, wantFFmpeg: 35},
		{name: "direct_play_resume", storedResume: 35, wantDirect: 35},
		{name: "skip_small_resume", storedResume: 9},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotFFmpeg, gotDirect := computeResumeStart(tc.existingSeek, tc.storedResume, tc.transcode)
			if gotFFmpeg != tc.wantFFmpeg || gotDirect != tc.wantDirect {
				t.Fatalf("got (%d,%d) want (%d,%d)", gotFFmpeg, gotDirect, tc.wantFFmpeg, tc.wantDirect)
			}
		})
	}
}

func TestComputeChromecastResumeStart(t *testing.T) {
	tests := []struct {
		name         string
		existingSeek int
		storedResume int
		want         int
	}{
		{name: "manual_seek_wins", existingSeek: 77, storedResume: 35, want: 77},
		{name: "resume_native_direct_play", storedResume: 35, want: 35},
		{name: "skip_small_resume", storedResume: 9, want: 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := computeChromecastResumeStart(tc.existingSeek, tc.storedResume)
			if got != tc.want {
				t.Fatalf("got %d want %d", got, tc.want)
			}
		})
	}
}

func TestShouldPersistResumePosition(t *testing.T) {
	now := time.Now()

	if shouldPersistResumePosition(9, 100, 0, time.Time{}, now, false) {
		t.Fatalf("expected near-zero position to skip")
	}

	if shouldPersistResumePosition(95, 100, 80, now.Add(-10*time.Second), now, false) {
		t.Fatalf("expected completed position to skip save")
	}

	if shouldPersistResumePosition(20, 100, 18, now.Add(-10*time.Second), now, false) {
		t.Fatalf("expected small delta to skip save")
	}

	if !shouldPersistResumePosition(20, 100, 10, now.Add(-10*time.Second), now, false) {
		t.Fatalf("expected meaningful change to save")
	}

	if !shouldPersistResumePosition(20, 100, 18, now, now, true) {
		t.Fatalf("expected force save to bypass cadence")
	}
}

func TestShouldRemoveResumeEntry(t *testing.T) {
	if !shouldRemoveResumeEntry(95, 100) {
		t.Fatalf("expected progress-based removal")
	}

	if !shouldRemoveResumeEntry(75, 100) {
		t.Fatalf("expected remaining-time removal")
	}

	if shouldRemoveResumeEntry(20, 100) {
		t.Fatalf("did not expect early removal")
	}
}

func TestCanonicalizeResumePathWindowsCaseInsensitive(t *testing.T) {
	path := `C:\Media\Movie.MP4`
	got := canonicalizeResumePath(path, "windows")
	if got != strings.ToLower(filepath.Clean(path)) {
		t.Fatalf("got %q", got)
	}
}

func TestResolveResumeIdentityResolvesSymlinks(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "track.mp3")
	if err := os.WriteFile(target, []byte("audio"), 0o600); err != nil {
		t.Fatalf("write file failed: %v", err)
	}

	link := filepath.Join(dir, "link.mp3")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}

	targetIdentity, ok, err := resolveResumeIdentity(target, "audio")
	if err != nil || !ok {
		t.Fatalf("target identity failed: %v ok=%v", err, ok)
	}

	linkIdentity, ok, err := resolveResumeIdentity(link, "audio")
	if err != nil || !ok {
		t.Fatalf("link identity failed: %v ok=%v", err, ok)
	}

	if linkIdentity != targetIdentity {
		t.Fatalf("expected symlink and target to match: %#v %#v", linkIdentity, targetIdentity)
	}
}

func TestResumeEntryMatchRejectsDifferentMtime(t *testing.T) {
	entry := resumeEntry{
		MediaPath:           "/tmp/example.mp4",
		Size:                10,
		MTimeUnixNano:       11,
		MediaKind:           "video",
		LastPositionSeconds: 22,
		UpdatedAtUnix:       33,
	}

	if resumeEntryMatches(entry, resumeIdentity{
		MediaPath:     "/tmp/example.mp4",
		Size:          10,
		MTimeUnixNano: 12,
		MediaKind:     "video",
	}) {
		t.Fatalf("expected mismatched mtime to fail")
	}
}

func TestResolveResumeIdentityMissingFile(t *testing.T) {
	_, ok, err := resolveResumeIdentity(filepath.Join(t.TempDir(), "missing.mp4"), "video")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if ok {
		t.Fatalf("expected missing file to return no identity")
	}
}

func TestShouldAttemptInitialDLNASeek(t *testing.T) {
	tests := []struct {
		name   string
		client fakeDLNAResumeClient
		want   bool
	}{
		{
			name: "playing_transport_ready",
			client: fakeDLNAResumeClient{
				transportInfo: []string{"PLAYING", "OK", "1"},
			},
			want: true,
		},
		{
			name: "paused_transport_ready",
			client: fakeDLNAResumeClient{
				transportInfo: []string{"PAUSED_PLAYBACK", "OK", "1"},
			},
			want: true,
		},
		{
			name: "transitioning_not_ready",
			client: fakeDLNAResumeClient{
				transportInfo: []string{"TRANSITIONING", "OK", "1"},
				positionInfo:  []string{"01:00:00", "00:00:00"},
			},
			want: false,
		},
		{
			name: "fallback_to_position_info",
			client: fakeDLNAResumeClient{
				transportErr: errors.New("unsupported"),
				positionInfo: []string{"01:00:00", "00:00:00"},
			},
			want: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldAttemptInitialDLNASeek(tc.client)
			if got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}
