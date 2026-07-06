package utils

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

type countingWriter struct {
	w io.Writer
	n int64
}

func (c *countingWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.n += int64(n)
	return n, err
}

func runFFmpegTranscode(
	ctx context.Context,
	ff *exec.Cmd,
	input any,
	in string,
	w io.Writer,
	args []string,
) (int64, error) {
	cmd := exec.Command(args[0], args[1:]...)
	setSysProcAttr(cmd)

	*ff = *cmd
	switch in {
	case "fd:":
		// The child reads fd 0 directly and shares its offset, so rewind to
		// keep retries (e.g. hardware -> software encoder) deterministic.
		f := input.(*os.File)
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return 0, fmt.Errorf("rewind ffmpeg input: %w", err)
		}
		ff.Stdin = f
	case "pipe:0":
		ff.Stdin = input.(io.Reader)
	}

	cw := &countingWriter{w: w}
	var stderr bytes.Buffer
	ff.Stdout = cw
	ff.Stderr = &stderr

	if err := ff.Start(); err != nil {
		return 0, fmt.Errorf("%w: %s", err, tailFFmpegStderr(strings.TrimSpace(stderr.String()), 240))
	}

	done := make(chan error, 1)
	go func() {
		done <- ff.Wait()
	}()

	select {
	case <-ctx.Done():
		if ff.Process != nil {
			_ = ff.Process.Kill()
		}
		<-done
		return cw.n, ctx.Err()
	case err := <-done:
		if err != nil {
			return cw.n, fmt.Errorf("%w: %s", err, tailFFmpegStderr(strings.TrimSpace(stderr.String()), 240))
		}
		return cw.n, nil
	}
}
