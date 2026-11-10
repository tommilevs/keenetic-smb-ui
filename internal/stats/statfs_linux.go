//go:build linux
// +build linux

package stats

import "syscall"

func diskUsage(path string) (Disk, bool) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return Disk{}, false
	}
	total := st.Blocks * uint64(st.Bsize)
	free := st.Bavail * uint64(st.Bsize)
	used := total - free
	d := Disk{Path: path, Total: total, Free: free, Used: used}
	if total > 0 {
		d.UsedPC = round2(float64(used) / float64(total) * 100)
	}
	return d, true
}
