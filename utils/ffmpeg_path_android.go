//go:build android

package utils

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const androidProcMaps = "/proc/self/maps"

func init() {
	platformFFmpegPath = androidBundledFFmpegPath
	platformFFprobePath = androidBundledFFprobePath
}

func androidBundledFFmpegPath() (string, error) {
	return androidBundledToolPath("libffmpeg.so")
}

func androidBundledFFprobePath() (string, error) {
	return androidBundledToolPath("libffprobe.so")
}

func androidBundledToolPath(name string) (string, error) {
	dir, err := androidNativeLibDir()
	if err != nil {
		return "", err
	}

	path := filepath.Join(dir, name)
	if isExistingFile(path) {
		return path, nil
	}

	return "", fmt.Errorf("%s: %w", path, exec.ErrNotFound)
}

func androidNativeLibDir() (string, error) {
	f, err := os.Open(androidProcMaps)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var fallback string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		for _, field := range strings.Fields(scanner.Text()) {
			if !androidAppNativeLibPath(field) {
				continue
			}

			dir := filepath.Dir(field)
			if strings.Contains(filepath.Base(field), "gojni") {
				return dir, nil
			}
			if fallback == "" {
				fallback = dir
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	if fallback != "" {
		return fallback, nil
	}

	return "", fmt.Errorf("android native lib dir: %w", exec.ErrNotFound)
}

func androidAppNativeLibPath(path string) bool {
	if !filepath.IsAbs(path) || strings.Contains(path, "!") {
		return false
	}

	base := filepath.Base(path)
	if !strings.HasPrefix(base, "lib") || !strings.HasSuffix(base, ".so") {
		return false
	}

	return strings.Contains(path, "/data/app/") &&
		(strings.Contains(path, "/lib/") || strings.Contains(path, "/lib64/"))
}
