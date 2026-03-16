package rtmp

import "strings"

func IsListenTimeoutError(err error) bool {
	if err == nil {
		return false
	}

	msg := err.Error()
	return strings.Contains(msg, "Connection timed out") &&
		strings.Contains(msg, "listen_timeout=") &&
		strings.Contains(msg, "rtmp://0.0.0.0:")
}
