// Package ndms talks to the Keenetic NDMS CLI (ndmc) to read router users.
package ndms

import (
	"bufio"
	"os/exec"
	"strings"
)

// User is a router account as reported by `ndmc -c "show user"`.
type User struct {
	Name     string `json:"name"`
	Readonly bool   `json:"readonly"`
	Manager  bool   `json:"manager"`
}

// Available reports whether the NDMS CLI is present on this device.
func Available() bool {
	_, err := exec.LookPath("ndmc")
	return err == nil
}

// ListUsers parses the NDMS user table.
func ListUsers() ([]User, error) {
	out, err := exec.Command("ndmc", "-c", "show user").Output()
	if err != nil {
		return nil, err
	}
	var (
		users   []User
		current *User
	)
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	for sc.Scan() {
		line := strings.TrimSpace(stripANSI(sc.Text()))
		if line == "" {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		switch key {
		case "user":
			if current != nil && current.Name != "" {
				users = append(users, *current)
			}
			current = &User{Name: value}
		case "readonly":
			if current != nil {
				current.Readonly = value == "yes"
			}
		case "manager":
			if current != nil {
				current.Manager = value == "yes"
			}
		}
	}
	if current != nil && current.Name != "" {
		users = append(users, *current)
	}
	return users, nil
}

// stripANSI removes terminal escape sequences that ndmc prints around output.
func stripANSI(s string) string {
	var b strings.Builder
	esc := false
	for _, r := range s {
		switch {
		case r == 0x1b:
			esc = true
		case esc:
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				esc = false
			}
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
