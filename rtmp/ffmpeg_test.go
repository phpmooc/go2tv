package rtmp

import (
	"errors"
	"testing"
)

func TestBuildCLICommandUsesSecondBasedListenTimeout(t *testing.T) {
	t.Parallel()

	args, err := BuildCLICommand("streamkey", "1935", "/tmp/go2tv-rtmp-test")
	if err != nil {
		t.Fatalf("BuildCLICommand() error = %v", err)
	}

	timeout, ok := flagValue(args, "-timeout")
	if !ok {
		t.Fatal("BuildCLICommand() missing -timeout flag")
	}

	if timeout != "300" {
		t.Fatalf("BuildCLICommand() timeout = %q, want %q", timeout, "300")
	}
}

func TestIsListenTimeoutError(t *testing.T) {
	t.Parallel()

	err := IsListenTimeoutError(errors.New("exit status 146: tcp://0.0.0.0:1935?listen&listen_timeout=300000 Connection timed out Error opening input file rtmp://0.0.0.0:1935/live/abc"))
	if !err {
		t.Fatal("IsListenTimeoutError() = false, want true")
	}

	err = IsListenTimeoutError(errors.New("exit status 1: some other ffmpeg failure"))
	if err {
		t.Fatal("IsListenTimeoutError() = true, want false")
	}
}
