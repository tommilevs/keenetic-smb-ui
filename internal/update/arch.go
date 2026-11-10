package update

import (
	"os/exec"
	"strings"
)

// DetectArch reports the Entware package architecture of this device, e.g.
// "aarch64-3.10". It prefers the architecture recorded for the installed
// package and falls back to uname-derived guesses.
func DetectArch() string {
	if out, err := exec.Command("opkg", "status", "keenetic-smb-ui").Output(); err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			if strings.HasPrefix(line, "Architecture:") {
				if v := strings.TrimSpace(strings.TrimPrefix(line, "Architecture:")); v != "" {
					return v
				}
			}
		}
	}
	uname, _ := exec.Command("uname", "-m").Output()
	switch strings.TrimSpace(string(uname)) {
	case "aarch64":
		return "aarch64-3.10"
	case "armv7l", "armv8l", "arm":
		return "armv7sf-k3.2"
	case "mips", "mipsle":
		if littleEndian() {
			return "mipselsf-k3.4"
		}
		return "mipssf-k3.4"
	case "x86_64":
		return "x64-64"
	default:
		return "i386sf-k3.2"
	}
}

func littleEndian() bool {
	out, err := exec.Command("sh", "-c", "printf '\\1\\0' | od -An -tu2").Output()
	if err != nil {
		return true
	}
	return strings.TrimSpace(string(out)) == "1" ||
		strings.HasPrefix(strings.TrimSpace(string(out)), "1")
}
