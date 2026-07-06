//go:build linux && !android

package gui

import (
	"strings"

	"golang.org/x/sys/unix"
)

const documentPortalHostPathXattr = "user.document-portal.host-path"

func queueDisplayPath(path string) string {
	size, err := unix.Getxattr(path, documentPortalHostPathXattr, nil)
	if err != nil || size <= 0 {
		return path
	}

	value := make([]byte, size)
	size, err = unix.Getxattr(path, documentPortalHostPathXattr, value)
	if err != nil || size <= 0 {
		return path
	}

	hostPath := strings.TrimRight(string(value[:size]), "\x00")
	if hostPath == "" {
		return path
	}

	return hostPath
}
