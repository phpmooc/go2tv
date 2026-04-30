package gui

import (
	"container/ring"
	"io"
	"sync"
)

const (
	runtimeDebugRingSize   = 1000
	discoveryDebugRingSize = 200
)

type debugWriter struct {
	mu   sync.Mutex
	ring *ring.Ring
}

func (f *debugWriter) Write(b []byte) (int, error) {
	if f == nil {
		return len(b), nil
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if f.ring == nil {
		return len(b), nil
	}

	f.ring.Value = string(b)
	f.ring = f.ring.Next()
	return len(b), nil
}

func newDebugWriter(size int) *debugWriter {
	if size <= 0 {
		size = 1
	}

	return &debugWriter{ring: ring.New(size)}
}

func hasDebugLogs(dw *debugWriter) bool {
	if dw == nil || dw.ring == nil {
		return false
	}

	dw.mu.Lock()
	defer dw.mu.Unlock()

	var itemInRing bool
	dw.ring.Do(func(p any) {
		if p != nil {
			itemInRing = true
		}
	})

	return itemInRing
}

func writeDebugLogs(w io.Writer, dw *debugWriter) error {
	if dw == nil || dw.ring == nil {
		return nil
	}

	dw.mu.Lock()
	defer dw.mu.Unlock()

	var writeErr error
	dw.ring.Do(func(p any) {
		if p == nil || writeErr != nil {
			return
		}

		_, writeErr = io.WriteString(w, p.(string))
	})

	return writeErr
}
