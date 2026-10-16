//go:build !windows

package api

import "path/filepath"

func resolveExistingDemoPath(path string) (string, error) {
	return filepath.EvalSymlinks(path)
}
