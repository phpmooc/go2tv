package utils

import (
	"net/url"
	"path"
	"strings"
)

// IsHLSStream returns true for HLS playlist URLs or HLS mime types.
func IsHLSStream(mediaURL, mediaType string) bool {
	trimmedURL := strings.TrimSpace(mediaURL)
	if trimmedURL != "" {
		u, err := url.Parse(trimmedURL)
		if err == nil && strings.EqualFold(path.Ext(u.Path), ".m3u8") {
			return true
		}
	}

	mime := strings.ToLower(strings.TrimSpace(mediaType))
	return strings.Contains(mime, "mpegurl")
}
