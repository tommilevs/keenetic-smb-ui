//go:build !linux
// +build !linux

package shares

import (
	"fmt"
	"os/exec"
	"strings"
)

// platformMount is the development fallback: it relies on the system mount
// helper. The real deployment target is Linux, where mount(2) is used.
func platformMount(source, target, data string) error {
	out, err := exec.Command("mount", "-t", "cifs", source, target, "-o", data).CombinedOutput()
	if err != nil {
		return fmt.Errorf("mount %s: %v: %s", source, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// ensureCIFS is a no-op outside Linux.
func ensureCIFS() error { return nil }
