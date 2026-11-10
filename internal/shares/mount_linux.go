//go:build linux
// +build linux

package shares

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// platformMount attaches a CIFS share using the mount(2) syscall directly.
// Entware-based Keenetic routers ship cifs.ko but no mount.cifs helper, so the
// syscall path is the only reliable way to mount without extra packages.
func platformMount(source, target, data string) error {
	if err := syscall.Mount(source, target, "cifs", 0, data); err != nil {
		return fmt.Errorf("mount %s: %w", source, err)
	}
	return nil
}

// ensureCIFS loads the cifs kernel module when the kernel has it as a module
// and it is not loaded yet.
func ensureCIFS() error {
	if fs, err := os.ReadFile("/proc/filesystems"); err == nil && bytes.Contains(fs, []byte("cifs")) {
		return nil
	}
	rel, err := exec.Command("uname", "-r").Output()
	if err != nil {
		return fmt.Errorf("uname: %w", err)
	}
	module := "/lib/modules/" + strings.TrimSpace(string(rel)) + "/cifs.ko"
	if _, err := os.Stat(module); err != nil {
		return fmt.Errorf("cifs not available in kernel and %s missing", module)
	}
	if out, err := exec.Command("insmod", module).CombinedOutput(); err != nil {
		return fmt.Errorf("insmod %s: %v: %s", module, err, strings.TrimSpace(string(out)))
	}
	return nil
}
