//go:build !linux
// +build !linux

package stats

// diskUsage is a stub for non-Linux builds (local development on Windows/macOS).
func diskUsage(path string) (Disk, bool) {
	return Disk{Path: path}, false
}
