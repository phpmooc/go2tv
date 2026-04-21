package rtmp

import (
	"errors"
	"strconv"
	"testing"
)

func TestBuildCLICommandUsesSecondBasedListenTimeout(t *testing.T) {
	t.Parallel()

	args := BuildCLICommand("streamkey", "1935", "/tmp/go2tv-rtmp-test")

	timeout, ok := flagValue(args, "-timeout")
	if !ok {
		t.Fatal("BuildCLICommand() missing -timeout flag")
	}

	want := strconv.Itoa(ListenTimeoutSeconds)
	if timeout != want {
		t.Fatalf("BuildCLICommand() timeout = %q, want %q", timeout, want)
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
