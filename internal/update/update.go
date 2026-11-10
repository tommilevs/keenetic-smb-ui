// Package update checks GitHub releases and installs a newer .ipk via opkg.
package update

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	repoAPI   = "https://api.github.com/repos/tommilevs/keenetic-smb-ui/releases/latest"
	archMatch = "aarch64-3.10" // overridden by caller; kept for reference
)

// Info describes the result of an update check.
type Info struct {
	Current string `json:"current"`
	Latest  string `json:"latest"`
	URL     string `json:"url,omitempty"`
	Notes   string `json:"notes,omitempty"`
	Newer   bool   `json:"newer"`
	Error   string `json:"error,omitempty"`
}

type release struct {
	TagName string `json:"tag_name"`
	Body    string `json:"body"`
	Assets  []struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
	} `json:"assets"`
}

// Check compares the running version with the latest GitHub release and picks
// the asset matching the package architecture (e.g. aarch64-3.10).
func Check(current, archSuffix string) Info {
	info := Info{Current: current}
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest(http.MethodGet, repoAPI, nil)
	if err != nil {
		info.Error = err.Error()
		return info
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "keenetic-smb-ui")
	resp, err := client.Do(req)
	if err != nil {
		info.Error = err.Error()
		return info
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		info.Error = fmt.Sprintf("github api: %s", resp.Status)
		return info
	}
	var rel release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		info.Error = err.Error()
		return info
	}
	info.Latest = strings.TrimPrefix(rel.TagName, "v")
	info.Notes = rel.Body
	info.Newer = compare(current, info.Latest) < 0
	for _, a := range rel.Assets {
		if strings.Contains(a.Name, archSuffix) && strings.HasSuffix(a.Name, ".ipk") {
			info.URL = a.URL
			break
		}
	}
	if info.Newer && info.URL == "" {
		info.Error = "в релизе нет пакета для этой архитектуры (" + archSuffix + ")"
	}
	return info
}

var versionRE = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)`)
func parseVersion(v string) (int, int, int) {
	m := versionRE.FindStringSubmatch(strings.TrimSpace(v))
	if m == nil {
		return 0, 0, 0
	}
	a, _ := strconv.Atoi(m[1])
	b, _ := strconv.Atoi(m[2])
	c, _ := strconv.Atoi(m[3])
	return a, b, c
}

func compare(a, b string) int {
	a1, a2, a3 := parseVersion(a)
	b1, b2, b3 := parseVersion(b)
	x := []int{a1, a2, a3}
	y := []int{b1, b2, b3}
	for i := range x {
		if x[i] < y[i] {
			return -1
		}
		if x[i] > y[i] {
			return 1
		}
	}
	return 0
}

// Install downloads url and installs it with opkg. The install runs detached
// because it replaces the running binary and restarts the service.
func Install(url string) error {
	script := fmt.Sprintf(`
sleep 2
cd /tmp || exit 1
curl -fsSL -o /tmp/keenetic-smb-ui.update.ipk %q || exit 1
opkg install --force-reinstall /tmp/keenetic-smb-ui.update.ipk || exit 1
/opt/etc/init.d/S99keenetic-smb-ui restart
rm -f /tmp/keenetic-smb-ui.update.ipk
`, url)
	cmd := exec.Command("/bin/sh", "-c", script)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
