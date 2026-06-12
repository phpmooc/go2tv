package utils

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// wrappedReadSeekCloser mirrors how refyne's URI readers embed the underlying
// reader and expose it via Unwrap.
type wrappedReadSeekCloser struct {
	io.ReadSeekCloser
}

func (w *wrappedReadSeekCloser) Unwrap() io.ReadSeekCloser {
	return w.ReadSeekCloser
}

func TestUnderlyingOSFile(t *testing.T) {
	f, err := os.Create(filepath.Join(t.TempDir(), "media.mp4"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	tests := []struct {
		name  string
		input io.Reader
		want  *os.File
		ok    bool
	}{
		{"bare file", f, f, true},
		{"wrapped file", &wrappedReadSeekCloser{ReadSeekCloser: f}, f, true},
		{"double wrapped file", &wrappedReadSeekCloser{ReadSeekCloser: &wrappedReadSeekCloser{ReadSeekCloser: f}}, f, true},
		{"plain reader", bytes.NewReader(nil), nil, false},
		{"wrapped nil", &wrappedReadSeekCloser{}, nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := underlyingOSFile(tt.input)
			if ok != tt.ok || got != tt.want {
				t.Fatalf("underlyingOSFile() = (%v, %v), want (%v, %v)", got, ok, tt.want, tt.ok)
			}
		})
	}
}
