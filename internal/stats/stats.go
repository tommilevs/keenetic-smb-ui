// Package stats samples router health: load, memory, network, disk usage and
// Samba session counters. Everything is stdlib-only so the binary stays small.
package stats

import (
	"bufio"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Point is one sample of the timeline shown in the UI.
type Point struct {
	TS      int64   `json:"ts"`
	Load1   float64 `json:"load1"`
	MemUsed float64 `json:"mem_used"` // percent
	RxKBps  float64 `json:"rx_kbps"`
	TxKBps  float64 `json:"tx_kbps"`
	Sessions int    `json:"sessions"`
}

// Disk is a mounted filesystem with usage.
type Disk struct {
	Source string  `json:"source"`
	Path   string  `json:"path"`
	Total  uint64  `json:"total"`
	Used   uint64  `json:"used"`
	Free   uint64  `json:"free"`
	UsedPC float64 `json:"used_pc"`
}

// Snapshot is the full current state returned to the UI.
type Snapshot struct {
	UptimeSec int64    `json:"uptime_sec"`
	Load      [3]float64 `json:"load"`
	MemTotal  uint64   `json:"mem_total"`
	MemFree   uint64   `json:"mem_free"`
	MemUsedPC float64  `json:"mem_used_pc"`
	RxKBps    float64  `json:"rx_kbps"`
	TxKBps    float64  `json:"tx_kbps"`
	Sessions  int      `json:"sessions"`
	Disks     []Disk   `json:"disks"`
	History   []Point  `json:"history"`
}

// Collector keeps previous counters for rate calculation.
type Collector struct {
	mu       sync.Mutex
	lastNet  netCounters
	lastTime time.Time
	history  []Point
	maxLen   int
	sharePaths []string
}

type netCounters struct{ rx, tx uint64 }

// New creates a collector; history length comes from the config.
func New(maxLen int, sharePaths []string) *Collector {
	if maxLen <= 0 {
		maxLen = 120
	}
	return &Collector{maxLen: maxLen, sharePaths: sharePaths}
}

// SetSharePaths updates the filesystems shown in the disks table.
func (c *Collector) SetSharePaths(paths []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sharePaths = paths
}

// Sample reads counters and current values.
func (c *Collector) Sample() Snapshot {
	c.mu.Lock()
	defer c.mu.Unlock()

	snap := Snapshot{}
	snap.UptimeSec = readUptime()
	snap.Load = readLoad()
	snap.MemTotal, snap.MemFree = readMem()
	if snap.MemTotal > 0 {
		snap.MemUsedPC = round2(float64(snap.MemTotal-snap.MemFree) / float64(snap.MemTotal) * 100)
	}

	now := time.Now()
	cur := readNet()
	if !c.lastTime.IsZero() {
		dt := now.Sub(c.lastTime).Seconds()
		if dt > 0 {
			snap.RxKBps = round2(float64(cur.rx-c.lastNet.rx) / 1024 / dt)
			snap.TxKBps = round2(float64(cur.tx-c.lastNet.tx) / 1024 / dt)
		}
	}
	c.lastNet = cur
	c.lastTime = now

	snap.Sessions = countSessions()
	snap.Disks = c.disksLocked()

	point := Point{
		TS:       now.Unix(),
		Load1:    snap.Load[0],
		MemUsed:  snap.MemUsedPC,
		RxKBps:   snap.RxKBps,
		TxKBps:   snap.TxKBps,
		Sessions: snap.Sessions,
	}
	c.history = append(c.history, point)
	if len(c.history) > c.maxLen {
		c.history = c.history[len(c.history)-c.maxLen:]
	}
	snap.History = append([]Point(nil), c.history...)
	return snap
}

// History returns the stored timeline without resampling.
func (c *Collector) History() []Point {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]Point(nil), c.history...)
}

func (c *Collector) disksLocked() []Disk {
	paths := append([]string(nil), c.sharePaths...)
	out := make([]Disk, 0, len(paths))
	for _, p := range paths {
		if d, ok := diskUsage(p); ok {
			out = append(out, d)
		}
	}
	return out
}

func readUptime() int64 {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return 0
	}
	v, _ := strconv.ParseFloat(fields[0], 64)
	return int64(v)
}

func readLoad() [3]float64 {
	var out [3]float64
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return out
	}
	fields := strings.Fields(string(data))
	for i := 0; i < 3 && i < len(fields); i++ {
		out[i], _ = strconv.ParseFloat(fields[i], 64)
	}
	return out
}

func readMem() (uint64, uint64) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0
	}
	defer f.Close()
	var total, avail uint64
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, "MemTotal:"):
			total = parseKB(line)
		case strings.HasPrefix(line, "MemAvailable:"):
			avail = parseKB(line)
		}
	}
	return total, avail
}

func parseKB(line string) uint64 {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return 0
	}
	v, _ := strconv.ParseUint(fields[1], 10, 64)
	return v * 1024
}

func readNet() netCounters {
	var c netCounters
	f, err := os.Open("/proc/net/dev")
	if err != nil {
		return c
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if !strings.Contains(line, ":") {
			continue
		}
		name := strings.TrimSpace(strings.SplitN(line, ":", 2)[0])
		if name == "lo" {
			continue
		}
		fields := strings.Fields(strings.SplitN(line, ":", 2)[1])
		if len(fields) < 9 {
			continue
		}
		rx, _ := strconv.ParseUint(fields[0], 10, 64)
		tx, _ := strconv.ParseUint(fields[8], 10, 64)
		c.rx += rx
		c.tx += tx
	}
	return c
}

func countSessions() int {
	out, err := exec.Command("smbstatus", "-P").Output()
	if err != nil {
		return 0
	}
	n := 0
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	for sc.Scan() {
		line := sc.Text()
		if strings.Contains(line, "PID") && strings.Contains(line, "Username") {
			continue
		}
		if strings.TrimSpace(line) != "" && len(strings.Fields(line)) >= 4 {
			n++
		}
	}
	return n
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}
