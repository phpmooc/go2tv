//go:build !linux || android

package gui

func queueDisplayPath(path string) string {
	return path
}
